package checkins

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

// HandleList handles GET /v1/checkins?profile_id=&from=&to=
func HandleList(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profileIDStr := r.URL.Query().Get("profile_id")
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")

		if profileIDStr == "" || from == "" || to == "" {
			writeError(w, http.StatusBadRequest, "missing_params", "profile_id, from, and to are required")
			return
		}

		profileID, err := uuid.Parse(profileIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_profile_id", "invalid profile_id format")
			return
		}

		checkins, err := service.ListCheckins(r.Context(), profileID, from, to)
		if err != nil {
			if errors.Is(err, ErrProfileNotFound) {
				writeError(w, http.StatusNotFound, "profile_not_found", err.Error())
				return
			}
			if errors.Is(err, ErrInvalidDate) {
				writeError(w, http.StatusBadRequest, "invalid_date", err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(CheckinsResponse{Checkins: checkins})
	}
}

// HandleUpsert handles POST /v1/checkins
func HandleUpsert(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req UpsertCheckinRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
			return
		}

		checkin, err := service.UpsertCheckin(r.Context(), req)
		if err != nil {
			if errors.Is(err, ErrProfileNotFound) {
				writeError(w, http.StatusNotFound, "profile_not_found", err.Error())
				return
			}
			if errors.Is(err, ErrInvalidType) {
				writeError(w, http.StatusBadRequest, "invalid_type", err.Error())
				return
			}
			if errors.Is(err, ErrInvalidScore) {
				writeError(w, http.StatusBadRequest, "invalid_score", err.Error())
				return
			}
			if errors.Is(err, ErrInvalidDate) {
				writeError(w, http.StatusBadRequest, "invalid_date", err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(checkin)
	}
}

// HandleDelete handles DELETE /v1/checkins/{id}
func HandleDelete(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		if idStr == "" {
			writeError(w, http.StatusBadRequest, "missing_id", "checkin id is required")
			return
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_id", "invalid checkin id format")
			return
		}

		if err := service.DeleteCheckin(r.Context(), id); err != nil {
			if errors.Is(err, ErrCheckinNotFound) {
				writeError(w, http.StatusNotFound, "checkin_not_found", err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
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
