package server

import (
	"errors"
	"net/http"

	"bob/internal/openflow"
	"bob/internal/policy"
	"bob/internal/protocol"
)

func (h Handler) handleOpen(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if !h.authorize(w, r) {
		return
	}

	var req protocol.OpenRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusInvalidRequest,
			Message: "invalid request body",
		})
		return
	}

	if resp := validateOpenEnvelope(req.Version, protocol.OpenVersionV1, req.Action); resp != nil {
		writeJSON(w, http.StatusBadRequest, *resp)
		return
	}

	result, err := h.openService.Open(r.Context(), openflow.Request{
		URL:            req.URL,
		LocalhostOnly:  h.config.LocalhostOnly,
		MirrorLoopback: false,
	})
	if err != nil {
		h.logger.Printf("open failed url=%s err=%v", policy.RedactForLog(req.URL), err)
		h.writeOpenError(w, result, err)
		return
	}

	h.logger.Printf("opened url=%s source_app=%s source_host=%s", policy.RedactForLog(result.OpenedURL), req.Source.App, req.Source.Host)
	writeJSON(w, http.StatusOK, protocol.OpenResponse{
		OK:     true,
		Status: protocol.StatusOpened,
	})
}

func (h Handler) handleOpenV2(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if !h.authorize(w, r) {
		return
	}

	var req protocol.OpenRequestV2
	if err := decodeJSONBody(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, protocol.OpenResponse{OK: false, Status: protocol.StatusInvalidRequest, Message: "invalid request body"})
		return
	}
	if resp := validateOpenEnvelope(req.Version, protocol.OpenVersionV2, req.Action); resp != nil {
		writeJSON(w, http.StatusBadRequest, *resp)
		return
	}
	result, err := h.openService.Open(r.Context(), openflow.Request{
		URL:            req.URL,
		Session:        req.Session,
		LocalhostOnly:  h.config.LocalhostOnly,
		MirrorLoopback: true,
	})
	if err != nil {
		h.logger.Printf("open failed session=%s url=%s err=%v", req.Session, policy.RedactForLog(req.URL), err)
		h.writeOpenError(w, result, err)
		return
	}

	resp := protocol.OpenResponse{
		OK:            true,
		Status:        protocol.StatusOpened,
		Rewritten:     result.Rewritten,
		LocalPort:     result.LocalPort,
		MappingReused: result.MappingReused,
	}
	if result.Rewritten {
		resp.OpenedURL = result.OpenedURL
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h Handler) writeOpenError(w http.ResponseWriter, result openflow.Result, err error) {
	var flowErr *openflow.Error
	if !errors.As(err, &flowErr) {
		writeJSON(w, http.StatusInternalServerError, protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusInternalError,
			Message: "internal error",
		})
		return
	}

	switch flowErr.Code {
	case openflow.CodeInvalidURL:
		writeJSON(w, http.StatusBadRequest, protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusInvalidURL,
			Message: flowErr.Message,
		})
	case openflow.CodeDenied:
		writeJSON(w, http.StatusForbidden, protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusDenied,
			Message: flowErr.Message,
		})
	case openflow.CodeSessionRequired:
		writeJSON(w, http.StatusBadRequest, protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusSessionRequired,
			Message: flowErr.Message,
		})
	case openflow.CodeSessionNotFound:
		writeJSON(w, http.StatusNotFound, protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusSessionNotFound,
			Message: flowErr.Message,
		})
	case openflow.CodeMirrorFailed:
		writeJSON(w, http.StatusInternalServerError, protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusMirrorFailed,
			Message: flowErr.Message,
		})
	case openflow.CodeOpenFailed:
		writeJSON(w, http.StatusInternalServerError, protocol.OpenResponse{
			OK:            false,
			Status:        protocol.StatusInternalError,
			Message:       flowErr.Message,
			OpenedURL:     result.OpenedURL,
			Rewritten:     result.Rewritten,
			LocalPort:     result.LocalPort,
			MappingReused: result.MappingReused,
		})
	default:
		writeJSON(w, http.StatusInternalServerError, protocol.OpenResponse{
			OK:      false,
			Status:  protocol.StatusInternalError,
			Message: "internal error",
		})
	}
}
