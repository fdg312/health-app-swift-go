package profiles

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// Handler содержит HTTP обработчики для профилей
type Handler struct {
	service *Service
}

// NewHandler создаёт новый handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// HandleList обрабатывает GET /v1/profiles
func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	profiles, err := h.service.ListProfiles(r.Context())
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "internal_error", "Failed to list profiles")
		return
	}

	h.sendJSON(w, http.StatusOK, ProfilesResponse{Profiles: profiles})
}

// HandleCreate обрабатывает POST /v1/profiles
func (h *Handler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	var req CreateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}

	profile, err := h.service.CreateProfile(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmptyName):
			h.sendError(w, http.StatusBadRequest, "empty_name", "Name cannot be empty")
		case errors.Is(err, ErrInvalidType):
			h.sendError(w, http.StatusBadRequest, "invalid_type", "Only 'guest' type is allowed")
		default:
			h.sendError(w, http.StatusInternalServerError, "internal_error", "Failed to create profile")
		}
		return
	}

	h.sendJSON(w, http.StatusCreated, profile)
}

// HandleUpdate обрабатывает PATCH /v1/profiles/{id}
func (h *Handler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		h.sendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	id, err := h.extractID(r.URL.Path)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_id", "Invalid profile ID")
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}

	profile, err := h.service.UpdateProfile(r.Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmptyName):
			h.sendError(w, http.StatusBadRequest, "empty_name", "Name cannot be empty")
		case errors.Is(err, ErrNotFound):
			h.sendError(w, http.StatusNotFound, "not_found", "Profile not found")
		default:
			h.sendError(w, http.StatusInternalServerError, "internal_error", "Failed to update profile")
		}
		return
	}

	h.sendJSON(w, http.StatusOK, profile)
}

// HandleDelete обрабатывает DELETE /v1/profiles/{id}
func (h *Handler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.sendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	id, err := h.extractID(r.URL.Path)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_id", "Invalid profile ID")
		return
	}

	err = h.service.DeleteProfile(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			h.sendError(w, http.StatusNotFound, "not_found", "Profile not found")
		case errors.Is(err, ErrCannotDeleteOwner):
			h.sendError(w, http.StatusConflict, "cannot_delete_owner", "Cannot delete owner profile")
		default:
			h.sendError(w, http.StatusInternalServerError, "internal_error", "Failed to delete profile")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// extractID извлекает UUID из пути /v1/profiles/{id}
func (h *Handler) extractID(path string) (uuid.UUID, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 {
		return uuid.Nil, errors.New("invalid path")
	}

	return uuid.Parse(parts[2])
}

// sendJSON отправляет JSON ответ
func (h *Handler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// sendError отправляет ошибку в формате ErrorResponse
func (h *Handler) sendError(w http.ResponseWriter, status int, code, message string) {
	h.sendJSON(w, status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
