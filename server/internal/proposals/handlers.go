package proposals

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
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

	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid limit")
			return
		}
		limit = parsed
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	resp, err := h.service.List(r.Context(), profileID, status, limit)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) HandleApply(w http.ResponseWriter, r *http.Request) {
	idRaw := strings.TrimSpace(r.PathValue("id"))
	proposalID, err := uuid.Parse(idRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid proposal id")
		return
	}

	resp, err := h.service.Apply(r.Context(), proposalID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) HandleReject(w http.ResponseWriter, r *http.Request) {
	idRaw := strings.TrimSpace(r.PathValue("id"))
	proposalID, err := uuid.Parse(idRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid proposal id")
		return
	}

	resp, err := h.service.Reject(r.Context(), proposalID)
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
	case errors.Is(err, ErrInvalidPayload):
		writeError(w, http.StatusBadRequest, "invalid_payload", "Invalid proposal payload")
	case errors.Is(err, ErrUnsupportedKind):
		writeError(w, http.StatusBadRequest, "unsupported_kind", "Proposal kind is not supported for apply")
	case errors.Is(err, ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
	case errors.Is(err, ErrProposalNotFound):
		writeError(w, http.StatusNotFound, "proposal_not_found", "Proposal not found")
	case errors.Is(err, ErrNotPending):
		writeError(w, http.StatusConflict, "not_pending", "Proposal is not pending")
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
