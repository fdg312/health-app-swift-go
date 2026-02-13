package workouts

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

// HandleGetPlan returns the active workout plan for a profile.
// GET /v1/workouts/plan?profile_id=<uuid>
func (h *Handlers) HandleGetPlan(w http.ResponseWriter, r *http.Request) {
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

	resp, err := h.service.GetActivePlan(r.Context(), profileID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleReplacePlan replaces the entire workout plan and items.
// PUT /v1/workouts/plan/replace
func (h *Handlers) HandleReplacePlan(w http.ResponseWriter, r *http.Request) {
	var req ReplaceItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_payload", "invalid JSON body")
		return
	}

	resp, err := h.service.ReplacePlanAndItems(r.Context(), &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleUpsertCompletion creates or updates a workout completion.
// POST /v1/workouts/completions
func (h *Handlers) HandleUpsertCompletion(w http.ResponseWriter, r *http.Request) {
	var req UpsertCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_payload", "invalid JSON body")
		return
	}

	resp, err := h.service.UpsertCompletion(r.Context(), &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleGetToday returns today's workout plan and completion status.
// GET /v1/workouts/today?profile_id=<uuid>&date=YYYY-MM-DD
func (h *Handlers) HandleGetToday(w http.ResponseWriter, r *http.Request) {
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

	date := strings.TrimSpace(r.URL.Query().Get("date"))
	if date == "" {
		// Default to today
		date = ""
	}

	resp, err := h.service.GetToday(r.Context(), profileID, date)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleListCompletions returns completions in a date range.
// GET /v1/workouts/completions?profile_id=<uuid>&from=YYYY-MM-DD&to=YYYY-MM-DD
func (h *Handlers) HandleListCompletions(w http.ResponseWriter, r *http.Request) {
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

	from := strings.TrimSpace(r.URL.Query().Get("from"))
	to := strings.TrimSpace(r.URL.Query().Get("to"))
	if from == "" || to == "" {
		writeError(w, http.StatusBadRequest, "invalid_range", "from and to dates are required")
		return
	}

	resp, err := h.service.ListCompletions(r.Context(), profileID, from, to)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// ============================================================================
// Error handling
// ============================================================================

func (h *Handlers) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
	case errors.Is(err, ErrProfileNotFound):
		writeError(w, http.StatusNotFound, "profile_not_found", "profile not found")
	case errors.Is(err, ErrPlanNotFound):
		writeError(w, http.StatusNotFound, "plan_not_found", "workout plan not found")
	case errors.Is(err, ErrItemNotFound):
		writeError(w, http.StatusNotFound, "item_not_found", "workout item not found")
	case errors.Is(err, ErrInvalidRequest):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

// ============================================================================
// Helpers
// ============================================================================

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

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
