package feed

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

// HandleGetDay handles GET /v1/feed/day?profile_id=&date=
func HandleGetDay(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profileIDStr := r.URL.Query().Get("profile_id")
		date := r.URL.Query().Get("date")

		if profileIDStr == "" || date == "" {
			writeError(w, http.StatusBadRequest, "missing_params", "profile_id and date are required")
			return
		}

		profileID, err := uuid.Parse(profileIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_profile_id", "invalid profile_id format")
			return
		}

		summary, err := service.GetDaySummary(r.Context(), profileID, date)
		if err != nil {
			if errors.Is(err, ErrProfileNotFound) {
				writeError(w, http.StatusNotFound, "profile_not_found", err.Error())
				return
			}
			if errors.Is(err, ErrInvalidDate) {
				writeError(w, http.StatusBadRequest, "invalid_date", err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(summary)
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
