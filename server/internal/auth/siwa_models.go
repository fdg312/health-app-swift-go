package auth

import "github.com/golang-jwt/jwt/v5"

// SIWAAuthRequest is the payload from iOS Sign in with Apple.
type SIWAAuthRequest struct {
	IdentityToken string `json:"identity_token"`
	User          string `json:"user,omitempty"`
	Email         string `json:"email,omitempty"`
	FullName      string `json:"full_name,omitempty"`
}

// SIWAAuthResponse is our local access token response after SIWA verification.
type SIWAAuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	UserID      string `json:"user_id"`
}

// AppleIdentityClaims are claims inside Apple identity token.
type AppleIdentityClaims struct {
	Email string `json:"email,omitempty"`
	jwt.RegisteredClaims
}
