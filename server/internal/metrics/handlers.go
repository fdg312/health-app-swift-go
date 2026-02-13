package metrics

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

// Handler содержит HTTP обработчики для метрик
type Handler struct {
	service *Service
}

// NewHandler создаёт новый handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// HandleSyncBatch обрабатывает POST /v1/sync/batch
func (h *Handler) HandleSyncBatch(w http.ResponseWriter, r *http.Request) {
	var req SyncBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}

	resp, err := h.service.SyncBatch(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrProfileNotFound):
			h.sendError(w, http.StatusNotFound, "profile_not_found", "Profile not found")
		case errors.Is(err, ErrInvalidDate):
			h.sendError(w, http.StatusBadRequest, "invalid_date", "Invalid date format")
		case errors.Is(err, ErrInvalidRange):
			h.sendError(w, http.StatusBadRequest, "invalid_range", "Invalid date range")
		case errors.Is(err, ErrInvalidStage):
			h.sendError(w, http.StatusBadRequest, "invalid_stage", "Invalid sleep stage")
		case errors.Is(err, ErrInvalidTime):
			h.sendError(w, http.StatusBadRequest, "invalid_time", "Invalid time range")
		default:
			h.sendError(w, http.StatusInternalServerError, "internal_error", "Failed to sync batch")
		}
		return
	}

	h.sendJSON(w, http.StatusOK, resp)
}

// HandleGetDailyMetrics обрабатывает GET /v1/metrics/daily
func (h *Handler) HandleGetDailyMetrics(w http.ResponseWriter, r *http.Request) {
	profileIDStr := r.URL.Query().Get("profile_id")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if profileIDStr == "" || from == "" || to == "" {
		h.sendError(w, http.StatusBadRequest, "missing_params", "Missing required parameters")
		return
	}

	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_profile_id", "Invalid profile ID")
		return
	}

	resp, err := h.service.GetDailyMetrics(r.Context(), profileID, from, to)
	if err != nil {
		switch {
		case errors.Is(err, ErrProfileNotFound):
			h.sendError(w, http.StatusNotFound, "profile_not_found", "Profile not found")
		case errors.Is(err, ErrInvalidDate):
			h.sendError(w, http.StatusBadRequest, "invalid_date", "Invalid date format")
		case errors.Is(err, ErrInvalidRange):
			h.sendError(w, http.StatusBadRequest, "invalid_range", "Invalid date range")
		default:
			h.sendError(w, http.StatusInternalServerError, "internal_error", "Failed to get daily metrics")
		}
		return
	}

	h.sendJSON(w, http.StatusOK, resp)
}

// HandleGetHourlyMetrics обрабатывает GET /v1/metrics/hourly
func (h *Handler) HandleGetHourlyMetrics(w http.ResponseWriter, r *http.Request) {
	profileIDStr := r.URL.Query().Get("profile_id")
	date := r.URL.Query().Get("date")
	metric := r.URL.Query().Get("metric")

	if profileIDStr == "" || date == "" || metric == "" {
		h.sendError(w, http.StatusBadRequest, "missing_params", "Missing required parameters")
		return
	}

	if metric != "steps" && metric != "hr" {
		h.sendError(w, http.StatusBadRequest, "invalid_metric", "Metric must be 'steps' or 'hr'")
		return
	}

	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_profile_id", "Invalid profile ID")
		return
	}

	resp, err := h.service.GetHourlyMetrics(r.Context(), profileID, date, metric)
	if err != nil {
		switch {
		case errors.Is(err, ErrProfileNotFound):
			h.sendError(w, http.StatusNotFound, "profile_not_found", "Profile not found")
		case errors.Is(err, ErrInvalidDate):
			h.sendError(w, http.StatusBadRequest, "invalid_date", "Invalid date format")
		default:
			h.sendError(w, http.StatusInternalServerError, "internal_error", "Failed to get hourly metrics")
		}
		return
	}

	h.sendJSON(w, http.StatusOK, resp)
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
