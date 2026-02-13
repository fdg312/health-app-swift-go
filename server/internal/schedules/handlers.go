package schedules

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type Handlers struct {
	service *Service
}

func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

func (h *Handlers) HandleList(w http.ResponseWriter, r *http.Request) {
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

	resp, err := h.service.List(r.Context(), profileID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handlers) HandleUpsert(w http.ResponseWriter, r *http.Request) {
	var req UpsertScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	resp, err := h.service.Upsert(r.Context(), req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handlers) HandleReplaceAll(w http.ResponseWriter, r *http.Request) {
	var req ReplaceSchedulesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	resp, err := h.service.ReplaceAll(r.Context(), req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handlers) HandleDelete(w http.ResponseWriter, r *http.Request) {
	idRaw := strings.TrimSpace(r.PathValue("id"))
	scheduleID, err := uuid.Parse(idRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid schedule id")
		return
	}

	if err := h.service.Delete(r.Context(), scheduleID); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidRequest):
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request")
	case errors.Is(err, ErrMaxSchedulesReached):
		writeError(w, http.StatusBadRequest, "invalid_request", "Maximum schedules per profile reached")
	case errors.Is(err, ErrSupplementNotFound):
		writeError(w, http.StatusBadRequest, "invalid_request", "Supplement not found")
	case errors.Is(err, ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
	case errors.Is(err, ErrProfileNotFound):
		writeError(w, http.StatusNotFound, "profile_not_found", "Profile not found")
	case errors.Is(err, ErrScheduleNotFound):
		writeError(w, http.StatusNotFound, "schedule_not_found", "Schedule not found")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{
		Error: ErrorDetail{Code: code, Message: message},
	})
}
