package chat

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) HandleListMessages(w http.ResponseWriter, r *http.Request) {
	profileIDRaw := strings.TrimSpace(r.URL.Query().Get("profile_id"))
	if profileIDRaw == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "profile_id is required")
		return
	}

	profileID, err := uuid.Parse(profileIDRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid profile_id")
		return
	}

	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid limit")
			return
		}
		limit = parsed
	}

	var before *time.Time
	if raw := strings.TrimSpace(r.URL.Query().Get("before")); raw != "" {
		parsed, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid before cursor")
			return
		}
		before = &parsed
	}

	resp, err := h.service.ListMessages(r.Context(), profileID, limit, before)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	resp, err := h.service.SendMessage(r.Context(), req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidRequest):
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request")
	case errors.Is(err, ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
	case errors.Is(err, ErrProfileNotFound):
		writeError(w, http.StatusNotFound, "profile_not_found", "Profile not found")
	case errors.Is(err, ErrAIFailed):
		writeError(w, http.StatusInternalServerError, "ai_failed", "AI provider failed")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
