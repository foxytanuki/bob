package server

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"bob/internal/config"
)

func TestHandleOpenUnauthorized(t *testing.T) {
	h := NewHandler(config.Daemon{Token: "secret", LocalhostOnly: true}, nil, log.New(&bytes.Buffer{}, "", 0))
	req := httptest.NewRequest(http.MethodPost, "/open", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: got %d want %d", rec.Code, http.StatusUnauthorized)
	}
}
