package nutrition

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

// Handler handles HTTP requests for nutrition targets.
type Handler struct {
	service *Service
}

// NewHandler creates a new nutrition handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// HandleGetTargets handles GET /v1/nutrition/targets?profile_id=
func (h *Handler) HandleGetTargets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ownerUserID := r.Context().Value("user_id").(string)

	// Parse profile_id query param
	profileIDStr := r.URL.Query().Get("profile_id")
	if profileIDStr == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "profile_id is required")
		return
	}

	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid profile_id format")
		return
	}

	// Get targets or defaults
	targets, isDefault, err := h.service.GetOrDefault(ctx, ownerUserID, profileID)
	if err != nil {
		if err.Error() == "profile_not_found" {
			writeError(w, http.StatusNotFound, "profile_not_found", "Profile not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to get nutrition targets")
		return
	}

	response := GetTargetsResponse{
		Targets:   targets,
		IsDefault: isDefault,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleUpsertTargets handles PUT /v1/nutrition/targets
func (h *Handler) HandleUpsertTargets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ownerUserID := r.Context().Value("user_id").(string)

	// Parse request body
	var req UpsertTargetsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_payload", "Invalid request body")
		return
	}

	// Upsert targets
	targets, err := h.service.Upsert(ctx, ownerUserID, req)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "profile_not_found" {
			writeError(w, http.StatusNotFound, "profile_not_found", "Profile not found")
			return
		}
		// Check if it's a validation error
		if len(errMsg) > 16 && errMsg[:16] == "invalid_request:" {
			writeError(w, http.StatusBadRequest, "invalid_request", errMsg[17:])
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to upsert nutrition targets")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(targets)
}

// writeError writes an error response in the standard format.
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
