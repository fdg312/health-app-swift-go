package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/fdg312/health-hub/internal/auth/emailotp"
)

type Handlers struct {
	service         *Service
	emailOTPService *emailotp.Service
}

func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

func (h *Handlers) WithEmailOTP(service *emailotp.Service) *Handlers {
	h.emailOTPService = service
	return h
}

// HandleSignInApple handles POST /v1/auth/apple
func (h *Handlers) HandleSignInApple(w http.ResponseWriter, r *http.Request) {
	var req SignInAppleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.IdentityToken == "" {
		writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "identity_token is required")
		return
	}

	resp, err := h.service.SignInWithApple(r.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "verify Apple token") {
			writeErrorResponse(w, http.StatusUnauthorized, "invalid_token", err.Error())
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// HandleDevAuth handles POST /v1/auth/dev
func (h *Handlers) HandleDevAuth(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.SignInDev(r.Context())
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func writeErrorResponse(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
