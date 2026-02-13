package notifications

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// HandleList handles GET /v1/inbox
func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
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

	onlyUnread := r.URL.Query().Get("only_unread") == "true"

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

	notifications, err := h.service.ListNotifications(r.Context(), profileID, onlyUnread, limit, offset)
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
	json.NewEncoder(w).Encode(InboxListResponse{Notifications: notifications})
}

// HandleUnreadCount handles GET /v1/inbox/unread-count
func (h *Handler) HandleUnreadCount(w http.ResponseWriter, r *http.Request) {
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

	count, err := h.service.UnreadCount(r.Context(), profileID)
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
	json.NewEncoder(w).Encode(UnreadCountResponse{Unread: count})
}

// HandleMarkRead handles POST /v1/inbox/mark-read
func (h *Handler) HandleMarkRead(w http.ResponseWriter, r *http.Request) {
	var req MarkReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if req.ProfileID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "profile_id is required")
		return
	}

	if len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "ids are required")
		return
	}

	marked, err := h.service.MarkRead(r.Context(), req.ProfileID, req.IDs)
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
	json.NewEncoder(w).Encode(MarkReadResponse{Marked: marked})
}

// HandleMarkAllRead handles POST /v1/inbox/mark-all-read
func (h *Handler) HandleMarkAllRead(w http.ResponseWriter, r *http.Request) {
	var req MarkAllReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if req.ProfileID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "profile_id is required")
		return
	}

	marked, err := h.service.MarkAllRead(r.Context(), req.ProfileID)
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
	json.NewEncoder(w).Encode(MarkAllReadResponse{Marked: marked})
}

// HandleGenerate handles POST /v1/inbox/generate
func (h *Handler) HandleGenerate(w http.ResponseWriter, r *http.Request) {
	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if req.ProfileID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "profile_id is required")
		return
	}

	if req.Date == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "date is required")
		return
	}

	resp, err := h.service.Generate(r.Context(), &req)
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
	json.NewEncoder(w).Encode(resp)
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
