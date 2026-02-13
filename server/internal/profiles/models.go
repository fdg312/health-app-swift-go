package profiles

import (
	"time"

	"github.com/google/uuid"
)

// ProfileDTO — DTO для API
type ProfileDTO struct {
	ID          uuid.UUID `json:"id"`
	OwnerUserID string    `json:"owner_user_id"`
	Type        string    `json:"type"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProfilesResponse — ответ для GET /v1/profiles
type ProfilesResponse struct {
	Profiles []ProfileDTO `json:"profiles"`
}

// CreateProfileRequest — запрос для POST /v1/profiles
type CreateProfileRequest struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// UpdateProfileRequest — запрос для PATCH /v1/profiles/{id}
type UpdateProfileRequest struct {
	Name string `json:"name"`
}

// ErrorResponse — формат ошибки
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
