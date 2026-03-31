package server

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bob/internal/config"
)

func TestHandleOpenUnauthorized(t *testing.T) {
	h := NewHandler(config.Daemon{Token: "secret", LocalhostOnly: true}, nil, nil, log.New(&bytes.Buffer{}, "", 0))
	req := httptest.NewRequest(http.MethodPost, "/open", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: got %d want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandleOpenV2MissingSession(t *testing.T) {
	h := NewHandler(config.Daemon{Token: "secret", LocalhostOnly: true}, nil, nil, log.New(&bytes.Buffer{}, "", 0))
	req := httptest.NewRequest(http.MethodPost, "/v2/open", strings.NewReader(`{"version":2,"action":"open_url","url":"http://127.0.0.1:8787"}`))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d want %d", rec.Code, http.StatusBadRequest)
	}
}
