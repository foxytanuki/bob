package openflow

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"time"

	"bob/internal/opener"
	"bob/internal/policy"
	"bob/internal/tunnel"
)

type Code string

const (
	CodeInvalidURL      Code = "INVALID_URL"
	CodeDenied          Code = "DENIED"
	CodeSessionRequired Code = "SESSION_REQUIRED"
	CodeSessionNotFound Code = "SESSION_NOT_FOUND"
	CodeMirrorFailed    Code = "MIRROR_FAILED"
	CodeOpenFailed      Code = "OPEN_FAILED"
	CodeInternal        Code = "INTERNAL"
)

const defaultOpenTimeout = 10 * time.Second

type Request struct {
	URL            string
	Session        string
	LocalhostOnly  bool
	MirrorLoopback bool
}

type Result struct {
	OpenedURL     string
	Rewritten     bool
	LocalPort     int
	MappingReused bool
}

type Error struct {
	Code    Code
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type MirrorManager interface {
	EnsureMirror(ctx context.Context, session string, remotePort int) (tunnel.Mapping, bool, error)
}

type Service struct {
	Opener      opener.Opener
	Mirror      MirrorManager
	OpenTimeout time.Duration
}

func (s Service) Open(ctx context.Context, req Request) (Result, error) {
	normalized, err := policy.NormalizeAndValidate(req.URL, req.LocalhostOnly)
	if err != nil {
		return Result{}, mapPolicyError(err)
	}

	result := Result{OpenedURL: normalized.String()}

	if req.MirrorLoopback && policy.IsLoopbackURL(normalized) {
		if req.Session == "" {
			return Result{}, &Error{
				Code:    CodeSessionRequired,
				Message: "session is required for loopback URLs",
			}
		}
		if s.Mirror == nil {
			return Result{}, &Error{
				Code:    CodeMirrorFailed,
				Message: "auto-mirror is unavailable on this daemon",
			}
		}

		mapping, reused, err := s.Mirror.EnsureMirror(ctx, req.Session, portFromURL(normalized))
		if err != nil {
			if errors.Is(err, tunnel.ErrSessionNotFound) {
				return Result{}, &Error{
					Code:    CodeSessionNotFound,
					Message: "session not found",
					Err:     err,
				}
			}
			return Result{}, &Error{
				Code:    CodeMirrorFailed,
				Message: "failed to ensure local mirror",
				Err:     err,
			}
		}

		result.OpenedURL = policy.RewriteLoopbackURL(normalized, mapping.LocalPort)
		result.Rewritten = true
		result.LocalPort = mapping.LocalPort
		result.MappingReused = reused
	}

	openCtx := ctx
	cancel := func() {}
	if timeout := s.timeout(); timeout > 0 {
		openCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	if err := s.Opener.Open(openCtx, result.OpenedURL); err != nil {
		return result, &Error{
			Code:    CodeOpenFailed,
			Message: "failed to open browser",
			Err:     err,
		}
	}

	return result, nil
}

func (s Service) timeout() time.Duration {
	if s.OpenTimeout > 0 {
		return s.OpenTimeout
	}
	return defaultOpenTimeout
}

func mapPolicyError(err error) error {
	switch {
	case errors.Is(err, policy.ErrInvalidURL):
		return &Error{
			Code:    CodeInvalidURL,
			Message: "invalid url",
			Err:     err,
		}
	case errors.Is(err, policy.ErrDeniedURL):
		return &Error{
			Code:    CodeDenied,
			Message: "url denied by policy",
			Err:     err,
		}
	default:
		return &Error{
			Code:    CodeInternal,
			Message: "policy error",
			Err:     err,
		}
	}
}

func portFromURL(parsedURL *url.URL) int {
	if parsedURL.Port() != "" {
		port, _ := strconv.Atoi(parsedURL.Port())
		return port
	}
	if parsedURL.Scheme == "https" {
		return 443
	}
	return 80
}
