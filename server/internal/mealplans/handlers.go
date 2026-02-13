package mealplans

import (
	"encoding/json"
	"net/http"
	"time"
)

// Handler handles HTTP requests for meal plans.
type Handler struct {
	service *Service
}

// NewHandler creates a new meal plans handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// HandleGet handles GET /v1/meal/plan?profile_id=
func (h *Handler) HandleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ownerUserID := r.Context().Value("user_id").(string)

	profileID := r.URL.Query().Get("profile_id")
	if profileID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "profile_id is required")
		return
	}

	plan, items, found, err := h.service.GetActive(ctx, ownerUserID, profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to get meal plan")
		return
	}

	if !found {
		// Return empty response when no active plan
		response := GetMealPlanResponse{
			Plan:  nil,
			Items: []MealPlanItemDTO{},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := GetMealPlanResponse{
		Plan:  plan,
		Items: items,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleReplace handles PUT /v1/meal/plan/replace
func (h *Handler) HandleReplace(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ownerUserID := r.Context().Value("user_id").(string)

	var req ReplaceMealPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_payload", "Invalid request body")
		return
	}

	plan, items, err := h.service.ReplaceActive(ctx, ownerUserID, req)
	if err != nil {
		errMsg := err.Error()
		if len(errMsg) > 19 && errMsg[:19] == "validation failed: " {
			writeError(w, http.StatusBadRequest, "invalid_request", errMsg[19:])
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to replace meal plan")
		return
	}

	response := GetMealPlanResponse{
		Plan:  plan,
		Items: items,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleGetToday handles GET /v1/meal/today?profile_id=&date=YYYY-MM-DD
func (h *Handler) HandleGetToday(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ownerUserID := r.Context().Value("user_id").(string)

	profileID := r.URL.Query().Get("profile_id")
	if profileID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "profile_id is required")
		return
	}

	dateStr := r.URL.Query().Get("date")
	items, err := h.service.GetToday(ctx, ownerUserID, profileID, dateStr)
	if err != nil {
		if err.Error() == "invalid date format, expected YYYY-MM-DD" {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to get today's meal plan")
		return
	}

	// Use actual date for response
	var responseDate string
	if dateStr == "" {
		responseDate = time.Now().UTC().Format("2006-01-02")
	} else {
		responseDate = dateStr
	}

	response := GetTodayResponse{
		Date:  responseDate,
		Items: items,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleDelete handles DELETE /v1/meal/plan?profile_id=
func (h *Handler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ownerUserID := r.Context().Value("user_id").(string)

	profileID := r.URL.Query().Get("profile_id")
	if profileID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "profile_id is required")
		return
	}

	err := h.service.DeleteActive(ctx, ownerUserID, profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete meal plan")
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
