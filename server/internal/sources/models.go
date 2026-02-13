package sources

import (
	"time"

	"github.com/google/uuid"
)

// CreateSourceRequest — запрос на создание source (link/note)
type CreateSourceRequest struct {
	ProfileID uuid.UUID  `json:"profile_id"`
	Kind      string     `json:"kind"` // "link" or "note"
	Title     *string    `json:"title,omitempty"`
	Text      *string    `json:"text,omitempty"`
	URL       *string    `json:"url,omitempty"`
	CheckinID *uuid.UUID `json:"checkin_id,omitempty"`
}

// SourceDTO — представление source для API
type SourceDTO struct {
	ID          uuid.UUID  `json:"id"`
	ProfileID   uuid.UUID  `json:"profile_id"`
	Kind        string     `json:"kind"`
	Title       *string    `json:"title,omitempty"`
	Text        *string    `json:"text,omitempty"`
	URL         *string    `json:"url,omitempty"`
	CheckinID   *uuid.UUID `json:"checkin_id,omitempty"`
	ContentType *string    `json:"content_type,omitempty"`
	SizeBytes   int64      `json:"size_bytes,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// SourcesResponse — список sources
type SourcesResponse struct {
	Sources []SourceDTO `json:"sources"`
}

// Constants
const (
	KindLink  = "link"
	KindNote  = "note"
	KindImage = "image"
)
