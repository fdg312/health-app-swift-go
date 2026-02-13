package checkins

import (
	"time"

	"github.com/google/uuid"
)

// Checkin types
const (
	TypeMorning = "morning"
	TypeEvening = "evening"
)

// Valid checkin types
var ValidTypes = []string{TypeMorning, TypeEvening}

// Score range
const (
	MinScore = 1
	MaxScore = 5
)

// Checkin represents a check-in entry
type Checkin struct {
	ID        uuid.UUID `json:"id"`
	ProfileID uuid.UUID `json:"profile_id"`
	Date      string    `json:"date"` // YYYY-MM-DD
	Type      string    `json:"type"` // "morning" or "evening"
	Score     int       `json:"score"`
	Tags      []string  `json:"tags"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CheckinDTO is the API response format
type CheckinDTO struct {
	ID        uuid.UUID `json:"id"`
	ProfileID uuid.UUID `json:"profile_id"`
	Date      string    `json:"date"`
	Type      string    `json:"type"`
	Score     int       `json:"score"`
	Tags      []string  `json:"tags,omitempty"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpsertCheckinRequest is the request body for creating/updating a check-in
type UpsertCheckinRequest struct {
	ProfileID uuid.UUID `json:"profile_id"`
	Date      string    `json:"date"`
	Type      string    `json:"type"`
	Score     int       `json:"score"`
	Tags      []string  `json:"tags,omitempty"`
	Note      string    `json:"note,omitempty"`
}

// CheckinsResponse is the response for listing check-ins
type CheckinsResponse struct {
	Checkins []CheckinDTO `json:"checkins"`
}

// ToDTO converts Checkin to CheckinDTO
func (c *Checkin) ToDTO() CheckinDTO {
	return CheckinDTO{
		ID:        c.ID,
		ProfileID: c.ProfileID,
		Date:      c.Date,
		Type:      c.Type,
		Score:     c.Score,
		Tags:      c.Tags,
		Note:      c.Note,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error code and message
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
