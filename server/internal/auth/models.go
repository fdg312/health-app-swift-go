package auth

import (
	"github.com/google/uuid"
)

// SignInAppleRequest — запрос на авторизацию через Apple
type SignInAppleRequest struct {
	IdentityToken string `json:"identity_token"`
}

// SignInAppleResponse — ответ на успешную авторизацию
type SignInAppleResponse struct {
	AccessToken    string    `json:"access_token"`
	OwnerUserID    string    `json:"owner_user_id"`
	OwnerProfileID uuid.UUID `json:"owner_profile_id"`
}

// DevAuthResponse — ответ на dev-авторизацию
type DevAuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// JWTClaims — claims для JWT token
type JWTClaims struct {
	Sub string `json:"sub"` // owner_user_id
	Iss string `json:"iss"` // issuer
	Exp int64  `json:"exp"` // expiration time
	Iat int64  `json:"iat"` // issued at
}

// AppleTokenClaims — claims из Apple identity token
type AppleTokenClaims struct {
	Sub   string `json:"sub"`   // unique user ID
	Email string `json:"email"` // email (optional)
	Aud   string `json:"aud"`   // audience
	Exp   int64  `json:"exp"`   // expiration
	Iat   int64  `json:"iat"`   // issued at
	Iss   string `json:"iss"`   // issuer (должен быть https://appleid.apple.com)
}

// ErrorResponse — формат ошибки
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
