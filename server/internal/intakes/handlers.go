package intakes

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Handlers struct {
	service *Service
}

func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// HandleCreateSupplement handles POST /v1/supplements
func (h *Handlers) HandleCreateSupplement(w http.ResponseWriter, r *http.Request) {
	var req CreateSupplementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if req.ProfileID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "profile_id is required")
		return
	}

	supplement, err := h.service.CreateSupplement(r.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "not_found") {
			writeError(w, http.StatusNotFound, "profile_not_found", err.Error())
			return
		}
		if strings.Contains(err.Error(), "max_supplements") {
			writeError(w, http.StatusBadRequest, "max_supplements_reached", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(supplement)
}

// HandleListSupplements handles GET /v1/supplements
func (h *Handlers) HandleListSupplements(w http.ResponseWriter, r *http.Request) {
	profileIDStr := r.URL.Query().Get("profile_id")
	if profileIDStr == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "profile_id is required")
		return
	}

	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid profile_id")
		return
	}

	supplements, err := h.service.ListSupplements(r.Context(), profileID)
	if err != nil {
		if strings.Contains(err.Error(), "profile_not_found") {
			writeError(w, http.StatusNotFound, "profile_not_found", "Profile not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SupplementsResponse{Supplements: supplements})
}

// HandleUpdateSupplement handles PATCH /v1/supplements/{id}
func (h *Handlers) HandleUpdateSupplement(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/v1/supplements/")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid supplement ID")
		return
	}

	var req UpdateSupplementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	supplement, err := h.service.UpdateSupplement(r.Context(), id, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not_found") {
			writeError(w, http.StatusNotFound, "supplement_not_found", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(supplement)
}

// HandleDeleteSupplement handles DELETE /v1/supplements/{id}
func (h *Handlers) HandleDeleteSupplement(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/v1/supplements/")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid supplement ID")
		return
	}

	if err := h.service.DeleteSupplement(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "supplement_not_found", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleAddWater handles POST /v1/intakes/water
func (h *Handlers) HandleAddWater(w http.ResponseWriter, r *http.Request) {
	var req AddWaterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if req.ProfileID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "profile_id is required")
		return
	}

	if err := h.service.AddWater(r.Context(), &req); err != nil {
		if strings.Contains(err.Error(), "not_found") {
			writeError(w, http.StatusNotFound, "profile_not_found", err.Error())
			return
		}
		if strings.Contains(err.Error(), "limit_exceeded") {
			writeError(w, http.StatusBadRequest, "daily_water_limit_exceeded", err.Error())
			return
		}
		if strings.Contains(err.Error(), "positive") {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// HandleGetIntakesDaily handles GET /v1/intakes/daily
func (h *Handlers) HandleGetIntakesDaily(w http.ResponseWriter, r *http.Request) {
	profileIDStr := r.URL.Query().Get("profile_id")
	if profileIDStr == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "profile_id is required")
		return
	}

	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid profile_id")
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	resp, err := h.service.GetIntakesDaily(r.Context(), profileID, date)
	if err != nil {
		if strings.Contains(err.Error(), "not_found") {
			writeError(w, http.StatusNotFound, "profile_not_found", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// HandleUpsertSupplementIntake handles POST /v1/intakes/supplements
func (h *Handlers) HandleUpsertSupplementIntake(w http.ResponseWriter, r *http.Request) {
	var req UpsertSupplementIntakeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if req.ProfileID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "profile_id is required")
		return
	}

	if req.SupplementID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "supplement_id is required")
		return
	}

	if err := h.service.UpsertSupplementIntake(r.Context(), &req); err != nil {
		if strings.Contains(err.Error(), "not_found") {
			writeError(w, http.StatusNotFound, "supplement_not_found", err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid") {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
