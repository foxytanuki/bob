package server

import (
	"encoding/json"
	"net/http"

	"bob/internal/protocol"
	"bob/internal/version"
)

func writeHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusInvalidRequest,
			Message: "method not allowed",
		})
		return
	}

	writeJSON(w, http.StatusOK, protocol.HealthResponse{
		OK:      true,
		Status:  protocol.StatusOK,
		Version: version.Version,
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}
