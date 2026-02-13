package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/fdg312/health-hub/internal/auth/emailotp"
)

type EmailOTPRequest struct {
	Email string `json:"email"`
}

type EmailOTPVerifyRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// HandleEmailOTPRequest handles POST /v1/auth/email/request.
func (h *Handlers) HandleEmailOTPRequest(w http.ResponseWriter, r *http.Request) {
	if h.emailOTPService == nil {
		writeErrorResponse(w, http.StatusNotFound, "email_auth_disabled", "Email auth is disabled")
		return
	}

	var req EmailOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "email is required")
		return
	}

	resp, err := h.emailOTPService.Request(r.Context(), req.Email)
	if err != nil {
		h.writeEmailOTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// HandleEmailOTPVerify handles POST /v1/auth/email/verify.
func (h *Handlers) HandleEmailOTPVerify(w http.ResponseWriter, r *http.Request) {
	if h.emailOTPService == nil {
		writeErrorResponse(w, http.StatusNotFound, "email_auth_disabled", "Email auth is disabled")
		return
	}

	var req EmailOTPVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Code) == "" {
		writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "email and code are required")
		return
	}

	resp, err := h.emailOTPService.Verify(r.Context(), req.Email, req.Code)
	if err != nil {
		h.writeEmailOTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) writeEmailOTPError(w http.ResponseWriter, err error) {
	var serviceErr *emailotp.ServiceError
	if errors.As(err, &serviceErr) {
		writeErrorResponse(w, serviceErr.Status, serviceErr.Code, serviceErr.Message)
		return
	}

	writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "Internal server error")
}
