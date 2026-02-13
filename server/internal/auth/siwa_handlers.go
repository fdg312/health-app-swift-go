package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// HandleSignInSIWA handles POST /v1/auth/siwa.
func (h *Handlers) HandleSignInSIWA(w http.ResponseWriter, r *http.Request) {
	var req SIWAAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if strings.TrimSpace(req.IdentityToken) == "" {
		writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "identity_token is required")
		return
	}

	resp, err := h.service.SignInSIWA(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIdentityToken):
			writeErrorResponse(w, http.StatusUnauthorized, "invalid_identity_token", "Invalid identity token")
		case errors.Is(err, ErrJWKSFetchFailed):
			writeErrorResponse(w, http.StatusInternalServerError, "jwks_fetch_failed", "Failed to fetch Apple JWKS")
		default:
			writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
