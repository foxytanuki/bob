package server

import (
	"encoding/json"
	"io"
	"net/http"

	"bob/internal/protocol"
)

func decodeJSONBody[T any](r *http.Request, target *T) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func validateOpenEnvelope(version, expected int, action string) *protocol.OpenResponse {
	if action != "" && action != protocol.ActionOpenURL {
		return &protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusInvalidRequest,
			Message: "unsupported action",
		}
	}
	if version != 0 && version != expected {
		return &protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusInvalidRequest,
			Message: "unsupported version",
		}
	}
	return nil
}
