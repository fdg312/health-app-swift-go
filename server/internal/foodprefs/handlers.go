package foodprefs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/fdg312/health-hub/internal/storage"
)

// Handler handles HTTP requests for food preferences.
type Handler struct {
	service *Service
}

// NewHandler creates a new food preferences handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// HandleList handles GET /v1/food/prefs?profile_id=&q=&limit=&offset=
func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ownerUserID := r.Context().Value("user_id").(string)

	// Parse query params
	profileID := r.URL.Query().Get("profile_id")
	if profileID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "profile_id is required")
		return
	}

	query := r.URL.Query().Get("q")
	limit := parseIntQuery(r, "limit", 50)
	offset := parseIntQuery(r, "offset", 0)

	// List food preferences
	prefs, total, err := h.service.List(ctx, ownerUserID, profileID, query, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list food preferences")
		return
	}

	// Convert to DTOs
	items := make([]FoodPrefDTO, len(prefs))
	for i, pref := range prefs {
		items[i] = toDTO(pref)
	}

	response := ListFoodPrefsResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleUpsert handles POST /v1/food/prefs
func (h *Handler) HandleUpsert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ownerUserID := r.Context().Value("user_id").(string)

	// Parse request body
	var req UpsertFoodPrefRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_payload", "Invalid request body")
		return
	}

	// Check for unknown fields
	r.Body.Close()

	// Upsert food preference
	pref, err := h.service.Upsert(ctx, ownerUserID, req.ProfileID, req)
	if err != nil {
		errMsg := err.Error()
		// Check for validation or business logic errors
		if len(errMsg) > 20 && errMsg[:20] == "validation failed: " {
			writeError(w, http.StatusBadRequest, "invalid_request", errMsg[20:])
			return
		}
		if errMsg == "food preference with this name already exists" {
			writeError(w, http.StatusConflict, "duplicate_name", errMsg)
			return
		}
		if errMsg == "maximum food preferences limit reached (200)" {
			writeError(w, http.StatusConflict, "limit_reached", errMsg)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to save food preference")
		return
	}

	dto := toDTO(pref)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dto)
}

// HandleDelete handles DELETE /v1/food/prefs/{id}
func (h *Handler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ownerUserID := r.Context().Value("user_id").(string)

	// Get ID from URL path - extract from /v1/food/prefs/{id}
	path := strings.TrimPrefix(r.URL.Path, "/v1/food/prefs/")
	id := strings.TrimSpace(path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "id is required")
		return
	}

	// Delete food preference
	err := h.service.Delete(ctx, ownerUserID, id)
	if err != nil {
		if err.Error() == "food preference not found or unauthorized" {
			writeError(w, http.StatusNotFound, "not_found", "Food preference not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete food preference")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// toDTO converts storage.FoodPref to FoodPrefDTO.
func toDTO(pref storage.FoodPref) FoodPrefDTO {
	return FoodPrefDTO{
		ID:              pref.ID,
		ProfileID:       pref.ProfileID,
		Name:            pref.Name,
		Tags:            pref.Tags,
		KcalPer100g:     pref.KcalPer100g,
		ProteinGPer100g: pref.ProteinGPer100g,
		FatGPer100g:     pref.FatGPer100g,
		CarbsGPer100g:   pref.CarbsGPer100g,
		CreatedAt:       pref.CreatedAt,
		UpdatedAt:       pref.UpdatedAt,
	}
}

// parseIntQuery parses an integer query parameter with a default value.
func parseIntQuery(r *http.Request, key string, defaultValue int) int {
	valStr := r.URL.Query().Get(key)
	if valStr == "" {
		return defaultValue
	}

	var val int
	if _, err := fmt.Sscanf(valStr, "%d", &val); err != nil {
		return defaultValue
	}

	return val
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
