package sources

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
)

// Handlers handles HTTP requests for sources
type Handlers struct {
	service *Service
}

// NewHandlers creates new handlers
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// HandleCreate handles POST /v1/sources (link/note)
func (h *Handlers) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	dto, err := h.service.CreateSource(r.Context(), req)
	if err != nil {
		switch err {
		case ErrProfileNotFound:
			writeError(w, http.StatusNotFound, "profile_not_found", "Profile not found")
		case ErrInvalidKind:
			writeError(w, http.StatusBadRequest, "invalid_kind", "Kind must be 'link' or 'note'")
		case ErrMissingURL:
			writeError(w, http.StatusBadRequest, "missing_url", "URL is required for link")
		case ErrMissingText:
			writeError(w, http.StatusBadRequest, "missing_text", "Text is required for note")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dto)
}

// HandleCreateImage handles POST /v1/sources/image (multipart upload)
func (h *Handlers) HandleCreateImage(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 32 MB in memory)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Failed to parse multipart form")
		return
	}

	// Get profile_id
	profileIDStr := r.FormValue("profile_id")
	if profileIDStr == "" {
		writeError(w, http.StatusBadRequest, "missing_profile_id", "profile_id is required")
		return
	}

	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_profile_id", "Invalid profile_id format")
		return
	}

	// Get optional checkin_id
	var checkinID *uuid.UUID
	if checkinIDStr := r.FormValue("checkin_id"); checkinIDStr != "" {
		cid, err := uuid.Parse(checkinIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_checkin_id", "Invalid checkin_id format")
			return
		}
		checkinID = &cid
	}

	// Get optional title
	var title *string
	if t := r.FormValue("title"); t != "" {
		title = &t
	}

	// Get file
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing_file", "File is required")
		return
	}
	file.Close() // Close immediately, service will reopen

	// Create image source
	dto, err := h.service.CreateImageSource(r.Context(), profileID, checkinID, title, fileHeader)
	if err != nil {
		switch err {
		case ErrProfileNotFound:
			writeError(w, http.StatusNotFound, "profile_not_found", "Profile not found")
		case ErrFileTooLarge:
			writeError(w, http.StatusBadRequest, "file_too_large", fmt.Sprintf("File exceeds maximum size of %d MB", h.service.maxUploadMB))
		case ErrUnsupportedMime:
			writeError(w, http.StatusBadRequest, "unsupported_mime", "File type not supported")
		case ErrMaxSourcesExceeded:
			writeError(w, http.StatusBadRequest, "max_sources_exceeded", fmt.Sprintf("Maximum %d sources per checkin", h.service.maxSourcesPerCheck))
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dto)
}

// HandleList handles GET /v1/sources
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

	// Optional query filter
	query := r.URL.Query().Get("query")

	// Optional checkin_id filter
	var checkinID *uuid.UUID
	if checkinIDStr := r.URL.Query().Get("checkin_id"); checkinIDStr != "" {
		cid, err := uuid.Parse(checkinIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_checkin_id", "Invalid checkin_id format")
			return
		}
		checkinID = &cid
	}

	// Pagination
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

	dtos, err := h.service.ListSources(r.Context(), profileID, query, checkinID, limit, offset)
	if err != nil {
		if err == ErrProfileNotFound {
			writeError(w, http.StatusNotFound, "profile_not_found", "Profile not found")
		} else {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SourcesResponse{Sources: dtos})
}

// HandleDownload handles GET /v1/sources/{id}/download
func (h *Handlers) HandleDownload(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	sourceID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid source ID")
		return
	}

	// Check if we should redirect (S3 mode) or serve directly (local mode)
	downloadURL, isRedirect, err := h.service.GetImageDownloadURL(r.Context(), sourceID)
	if err != nil {
		if err == ErrSourceNotFound {
			writeError(w, http.StatusNotFound, "source_not_found", "Source not found")
		} else {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		}
		return
	}

	// If S3 mode, redirect to presigned or public URL
	if isRedirect {
		http.Redirect(w, r, downloadURL, http.StatusFound)
		return
	}

	// Local mode: serve file directly
	data, contentType, err := h.service.GetImageData(r.Context(), sourceID)
	if err != nil {
		if err == ErrSourceNotFound {
			writeError(w, http.StatusNotFound, "source_not_found", "Source not found")
		} else {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		}
		return
	}

	// Determine file extension from content type
	ext := ".jpg"
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/jpeg":
		ext = ".jpg"
	case "image/heic":
		ext = ".heic"
	}

	filename := fmt.Sprintf("source_%s%s", sourceID.String()[:8], ext)

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s", filepath.Base(filename)))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}

// HandleDelete handles DELETE /v1/sources/{id}
func (h *Handlers) HandleDelete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	sourceID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid source ID")
		return
	}

	if err := h.service.DeleteSource(r.Context(), sourceID); err != nil {
		if err == ErrSourceNotFound {
			writeError(w, http.StatusNotFound, "source_not_found", "Source not found")
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
