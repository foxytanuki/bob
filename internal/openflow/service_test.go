package openflow

import (
	"context"
	"errors"
	"testing"

	"bob/internal/tunnel"
)

type fakeOpener struct {
	urls []string
	err  error
}

func (f *fakeOpener) Open(ctx context.Context, rawURL string) error {
	f.urls = append(f.urls, rawURL)
	return f.err
}

type fakeMirror struct {
	mapping tunnel.Mapping
	reused  bool
	err     error
	calls   []struct {
		session string
		port    int
	}
}

func (f *fakeMirror) EnsureMirror(ctx context.Context, session string, remotePort int) (tunnel.Mapping, bool, error) {
	f.calls = append(f.calls, struct {
		session string
		port    int
	}{session: session, port: remotePort})
	return f.mapping, f.reused, f.err
}

func TestOpenRejectsInvalidURL(t *testing.T) {
	svc := Service{Opener: &fakeOpener{}}

	_, err := svc.Open(context.Background(), Request{URL: "mailto:test@example.com"})
	var flowErr *Error
	if !errors.As(err, &flowErr) {
		t.Fatalf("Open() error = %v, want *Error", err)
	}
	if flowErr.Code != CodeInvalidURL {
		t.Fatalf("code = %q, want %q", flowErr.Code, CodeInvalidURL)
	}
}

func TestOpenRejectsDeniedURL(t *testing.T) {
	svc := Service{Opener: &fakeOpener{}}

	_, err := svc.Open(context.Background(), Request{
		URL:           "https://example.com",
		LocalhostOnly: true,
	})
	var flowErr *Error
	if !errors.As(err, &flowErr) {
		t.Fatalf("Open() error = %v, want *Error", err)
	}
	if flowErr.Code != CodeDenied {
		t.Fatalf("code = %q, want %q", flowErr.Code, CodeDenied)
	}
}

func TestOpenRequiresSessionForLoopbackMirror(t *testing.T) {
	svc := Service{Opener: &fakeOpener{}}

	_, err := svc.Open(context.Background(), Request{
		URL:            "http://127.0.0.1:8787",
		MirrorLoopback: true,
	})
	var flowErr *Error
	if !errors.As(err, &flowErr) {
		t.Fatalf("Open() error = %v, want *Error", err)
	}
	if flowErr.Code != CodeSessionRequired {
		t.Fatalf("code = %q, want %q", flowErr.Code, CodeSessionRequired)
	}
}

func TestOpenFailsWhenMirrorUnavailable(t *testing.T) {
	svc := Service{Opener: &fakeOpener{}}

	_, err := svc.Open(context.Background(), Request{
		URL:            "http://127.0.0.1:8787",
		Session:        "demo",
		MirrorLoopback: true,
	})
	var flowErr *Error
	if !errors.As(err, &flowErr) {
		t.Fatalf("Open() error = %v, want *Error", err)
	}
	if flowErr.Code != CodeMirrorFailed {
		t.Fatalf("code = %q, want %q", flowErr.Code, CodeMirrorFailed)
	}
}

func TestOpenRewritesLoopbackURLBeforeOpening(t *testing.T) {
	op := &fakeOpener{}
	mirror := &fakeMirror{
		mapping: tunnel.Mapping{LocalPort: 43210},
		reused:  true,
	}
	svc := Service{
		Opener: op,
		Mirror: mirror,
	}

	result, err := svc.Open(context.Background(), Request{
		URL:            "http://127.0.0.1:8787/path?a=1",
		Session:        "demo",
		MirrorLoopback: true,
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if len(mirror.calls) != 1 || mirror.calls[0].session != "demo" || mirror.calls[0].port != 8787 {
		t.Fatalf("mirror calls = %#v", mirror.calls)
	}
	if len(op.urls) != 1 || op.urls[0] != "http://127.0.0.1:43210/path?a=1" {
		t.Fatalf("opened URLs = %#v", op.urls)
	}
	if !result.Rewritten || !result.MappingReused || result.LocalPort != 43210 {
		t.Fatalf("result = %#v", result)
	}
}

func TestOpenMapsMissingSessionFromMirror(t *testing.T) {
	op := &fakeOpener{}
	mirror := &fakeMirror{err: tunnel.ErrSessionNotFound}
	svc := Service{
		Opener: op,
		Mirror: mirror,
	}

	_, err := svc.Open(context.Background(), Request{
		URL:            "http://127.0.0.1:8787",
		Session:        "demo",
		MirrorLoopback: true,
	})
	var flowErr *Error
	if !errors.As(err, &flowErr) {
		t.Fatalf("Open() error = %v, want *Error", err)
	}
	if flowErr.Code != CodeSessionNotFound {
		t.Fatalf("code = %q, want %q", flowErr.Code, CodeSessionNotFound)
	}
	if len(op.urls) != 0 {
		t.Fatalf("opener should not be called, got %#v", op.urls)
	}
}

func TestOpenReturnsPartialResultOnOpenFailure(t *testing.T) {
	op := &fakeOpener{err: errors.New("boom")}
	mirror := &fakeMirror{
		mapping: tunnel.Mapping{LocalPort: 43210},
	}
	svc := Service{
		Opener: op,
		Mirror: mirror,
	}

	result, err := svc.Open(context.Background(), Request{
		URL:            "http://127.0.0.1:8787",
		Session:        "demo",
		MirrorLoopback: true,
	})
	var flowErr *Error
	if !errors.As(err, &flowErr) {
		t.Fatalf("Open() error = %v, want *Error", err)
	}
	if flowErr.Code != CodeOpenFailed {
		t.Fatalf("code = %q, want %q", flowErr.Code, CodeOpenFailed)
	}
	if result.OpenedURL != "http://127.0.0.1:43210" || !result.Rewritten || result.LocalPort != 43210 {
		t.Fatalf("result = %#v", result)
	}
}
