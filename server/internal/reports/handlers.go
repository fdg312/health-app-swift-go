package reports

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

// Handlers handles HTTP requests for reports
type Handlers struct {
	service *Service
}

// NewHandlers creates new handlers
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// HandleCreate handles POST /v1/reports
func (h *Handlers) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	report, err := h.service.CreateReport(r.Context(), req)
	if err != nil {
		switch err {
		case ErrInvalidFormat:
			writeError(w, http.StatusBadRequest, "invalid_format", "Format must be 'pdf' or 'csv'")
		case ErrInvalidDate:
			writeError(w, http.StatusBadRequest, "invalid_date", "Invalid date format, use YYYY-MM-DD")
		case ErrInvalidDateRange:
			writeError(w, http.StatusBadRequest, "invalid_range", "From date must be before to date")
		case ErrRangeTooLarge:
			writeError(w, http.StatusBadRequest, "range_too_large", fmt.Sprintf("Date range exceeds maximum of %d days", h.service.maxRangeDays))
		case ErrProfileNotFound:
			writeError(w, http.StatusNotFound, "profile_not_found", "Profile not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		}
		return
	}

	// Generate download URL
	baseURL := getBaseURL(r)
	downloadURL, err := h.service.GetReportDownloadURL(r.Context(), report.ID, baseURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to generate download URL")
		return
	}

	// Build response
	dto := ReportDTO{
		ID:          report.ID,
		ProfileID:   report.ProfileID,
		Format:      report.Format,
		From:        report.FromDate,
		To:          report.ToDate,
		DownloadURL: downloadURL,
		SizeBytes:   report.SizeBytes,
		Status:      report.Status,
		CreatedAt:   report.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dto)
}

// HandleList handles GET /v1/reports
func (h *Handlers) HandleList(w http.ResponseWriter, r *http.Request) {
	profileIDStr := r.URL.Query().Get("profile_id")
	if profileIDStr == "" {
		writeError(w, http.StatusBadRequest, "missing_profile_id", "profile_id is required")
		return
	}

	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_profile_id", "Invalid profile_id format")
		return
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	reports, err := h.service.ListReports(r.Context(), profileID, limit, offset)
	if err != nil {
		if err == ErrProfileNotFound {
			writeError(w, http.StatusNotFound, "profile_not_found", "Profile not found")
		} else {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		}
		return
	}

	// Build response
	baseURL := getBaseURL(r)
	dtos := make([]ReportDTO, len(reports))
	for i, report := range reports {
		downloadURL, _ := h.service.GetReportDownloadURL(r.Context(), report.ID, baseURL)
		dtos[i] = ReportDTO{
			ID:          report.ID,
			ProfileID:   report.ProfileID,
			Format:      report.Format,
			From:        report.FromDate,
			To:          report.ToDate,
			DownloadURL: downloadURL,
			SizeBytes:   report.SizeBytes,
			Status:      report.Status,
			CreatedAt:   report.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ReportsResponse{Reports: dtos})
}

// HandleDownload handles GET /v1/reports/{id}/download
func (h *Handlers) HandleDownload(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	reportID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid report ID")
		return
	}

	// Get report metadata
	report, err := h.service.GetReport(r.Context(), reportID)
	if err != nil {
		if err == ErrReportNotFound {
			writeError(w, http.StatusNotFound, "report_not_found", "Report not found")
		} else {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		}
		return
	}

	// Check if local mode or S3 mode
	if h.service.localMode {
		// Local mode: serve file directly
		data, contentType, err := h.service.GetReportData(r.Context(), reportID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		filename := fmt.Sprintf("report_%s_%s.%s", report.FromDate, report.ToDate, report.Format)
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		w.Header().Set("Content-Length", strconv.FormatInt(int64(len(data)), 10))
		w.Write(data)
	} else {
		// S3 mode: redirect to presigned URL
		baseURL := getBaseURL(r)
		presignedURL, err := h.service.GetReportDownloadURL(r.Context(), reportID, baseURL)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "Failed to generate download URL")
			return
		}

		http.Redirect(w, r, presignedURL, http.StatusFound)
	}
}

// HandleDelete handles DELETE /v1/reports/{id}
func (h *Handlers) HandleDelete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	reportID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid report ID")
		return
	}

	if err := h.service.DeleteReport(r.Context(), reportID); err != nil {
		if err == ErrReportNotFound {
			writeError(w, http.StatusNotFound, "report_not_found", "Report not found")
		} else {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

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

func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := r.Host
	return fmt.Sprintf("%s://%s", scheme, host)
}
