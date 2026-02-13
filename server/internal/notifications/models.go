package notifications

import (
	"time"

	"github.com/google/uuid"
)

// NotificationDTO — DTO для уведомления
type NotificationDTO struct {
	ID         uuid.UUID  `json:"id"`
	ProfileID  uuid.UUID  `json:"profile_id"`
	Kind       string     `json:"kind"`
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	SourceDate *string    `json:"source_date,omitempty"` // YYYY-MM-DD
	Severity   string     `json:"severity"`
	CreatedAt  time.Time  `json:"created_at"`
	ReadAt     *time.Time `json:"read_at,omitempty"`
}

// InboxListResponse — ответ для GET /v1/inbox
type InboxListResponse struct {
	Notifications []NotificationDTO `json:"notifications"`
}

// UnreadCountResponse — ответ для GET /v1/inbox/unread-count
type UnreadCountResponse struct {
	Unread int `json:"unread"`
}

// MarkReadRequest — запрос для POST /v1/inbox/mark-read
type MarkReadRequest struct {
	ProfileID uuid.UUID   `json:"profile_id"`
	IDs       []uuid.UUID `json:"ids"`
}

// MarkReadResponse — ответ для POST /v1/inbox/mark-read
type MarkReadResponse struct {
	Marked int `json:"marked"`
}

// MarkAllReadRequest — запрос для POST /v1/inbox/mark-all-read
type MarkAllReadRequest struct {
	ProfileID uuid.UUID `json:"profile_id"`
}

// MarkAllReadResponse — ответ для POST /v1/inbox/mark-all-read
type MarkAllReadResponse struct {
	Marked int `json:"marked"`
}

// GenerateRequest — запрос для POST /v1/inbox/generate
type GenerateRequest struct {
	ProfileID      uuid.UUID          `json:"profile_id"`
	Date           string             `json:"date"` // YYYY-MM-DD
	ClientTimeZone string             `json:"client_time_zone"`
	Now            time.Time          `json:"now"` // RFC3339
	Thresholds     GenerateThresholds `json:"thresholds"`
}

type GenerateThresholds struct {
	SleepMinMinutes      int `json:"sleep_min_minutes"`
	StepsMin             int `json:"steps_min"`
	ActiveEnergyMinKcal  int `json:"active_energy_min_kcal"`
}

// GenerateResponse — ответ для POST /v1/inbox/generate
type GenerateResponse struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
}

// ErrorResponse — формат ошибки
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
