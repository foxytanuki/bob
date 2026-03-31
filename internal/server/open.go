package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"bob/internal/auth"
	"bob/internal/config"
	"bob/internal/opener"
	"bob/internal/policy"
	"bob/internal/protocol"
)

type Handler struct {
	config config.Daemon
	opener opener.Opener
	logger *log.Logger
}

func NewHandler(cfg config.Daemon, op opener.Opener, logger *log.Logger) http.Handler {
	h := Handler{
		config: cfg,
		opener: op,
		logger: logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", writeHealth)
	mux.HandleFunc("/open", h.handleOpen)
	return mux
}

func (h Handler) handleOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusInvalidRequest,
			Message: "method not allowed",
		})
		return
	}

	if !auth.ValidateBearerToken(r.Header.Get("Authorization"), h.config.Token) {
		h.logger.Printf("deny unauthorized request from %s", r.RemoteAddr)
		writeJSON(w, http.StatusUnauthorized, protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusUnauthorized,
			Message: "invalid or missing bearer token",
		})
		return
	}

	var req protocol.OpenRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusInvalidRequest,
			Message: "invalid request body",
		})
		return
	}

	if req.Action != "" && req.Action != protocol.ActionOpenURL {
		writeJSON(w, http.StatusBadRequest, protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusInvalidRequest,
			Message: "unsupported action",
		})
		return
	}

	if req.Version != 0 && req.Version != protocol.CurrentVersion {
		writeJSON(w, http.StatusBadRequest, protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusInvalidRequest,
			Message: "unsupported version",
		})
		return
	}

	normalized, err := policy.NormalizeAndValidate(req.URL, h.config.LocalhostOnly)
	if err != nil {
		h.writePolicyError(w, err, req.URL)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := h.opener.Open(ctx, normalized.String()); err != nil {
		h.logger.Printf("open failed url=%s err=%v", policy.RedactForLog(normalized.String()), err)
		writeJSON(w, http.StatusInternalServerError, protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusInternalError,
			Message: "failed to open browser",
		})
		return
	}

	h.logger.Printf("opened url=%s source_app=%s source_host=%s", policy.RedactForLog(normalized.String()), req.Source.App, req.Source.Host)
	writeJSON(w, http.StatusOK, protocol.OpenResponse{
		OK:     true,
		Status: protocol.StatusOpened,
	})
}

func (h Handler) writePolicyError(w http.ResponseWriter, err error, rawURL string) {
	sanitized := policy.RedactForLog(rawURL)
	if errors.Is(err, policy.ErrDeniedURL) {
		h.logger.Printf("deny policy url=%s", sanitized)
		writeJSON(w, http.StatusForbidden, protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusDenied,
			Message: "url denied by policy",
		})
		return
	}

	if errors.Is(err, policy.ErrInvalidURL) {
		h.logger.Printf("deny invalid url=%s", sanitized)
		writeJSON(w, http.StatusBadRequest, protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusInvalidURL,
			Message: "invalid url",
		})
		return
	}

	h.logger.Printf("unexpected policy error url=%s err=%v", sanitized, err)
	writeJSON(w, http.StatusInternalServerError, protocol.OpenResponse{
		OK:      false,
		Status:  protocol.StatusInternalError,
		Message: fmt.Sprintf("policy error: %v", err),
	})
}
