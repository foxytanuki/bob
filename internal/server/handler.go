package server

import (
	"io"
	"log"
	"net/http"

	"bob/internal/auth"
	"bob/internal/config"
	"bob/internal/opener"
	"bob/internal/openflow"
	"bob/internal/protocol"
)

type Handler struct {
	config      config.Daemon
	openService openflow.Service
	logger      *log.Logger
}

func NewHandler(cfg config.Daemon, op opener.Opener, mgr openflow.MirrorManager, logger *log.Logger) http.Handler {
	h := Handler{
		config: cfg,
		openService: openflow.Service{
			Opener: op,
			Mirror: mgr,
		},
		logger: newLogger(logger),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", writeHealth)
	mux.HandleFunc("/open", h.handleOpen)
	mux.HandleFunc("/v2/open", h.handleOpenV2)
	return mux
}

func newLogger(logger *log.Logger) *log.Logger {
	if logger != nil {
		return logger
	}
	return log.New(io.Discard, "", 0)
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	writeJSON(w, http.StatusMethodNotAllowed, protocol.OpenResponse{
		OK:      false,
		Status:  protocol.StatusInvalidRequest,
		Message: "method not allowed",
	})
	return false
}

func (h Handler) authorize(w http.ResponseWriter, r *http.Request) bool {
	if auth.ValidateBearerToken(r.Header.Get("Authorization"), h.config.Token) {
		return true
	}

	h.logger.Printf("deny unauthorized request from %s", r.RemoteAddr)
	writeJSON(w, http.StatusUnauthorized, protocol.OpenResponse{
		OK:      false,
		Status:  protocol.StatusUnauthorized,
		Message: "invalid or missing bearer token",
	})
	return false
}
