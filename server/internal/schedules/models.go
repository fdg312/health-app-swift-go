package schedules

import (
	"fmt"
	"strings"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

const (
	maxSchedulesPerProfile = 20
)

// days_mask mapping: bit 0 = Monday ... bit 6 = Sunday.

type ScheduleDTO struct {
	ID           uuid.UUID `json:"id"`
	SupplementID uuid.UUID `json:"supplement_id"`
	TimeMinutes  int       `json:"time_minutes"`
	DaysMask     int       `json:"days_mask"`
	IsEnabled    bool      `json:"is_enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ListSchedulesResponse struct {
	Schedules []ScheduleDTO `json:"schedules"`
}

type UpsertScheduleRequest struct {
	ProfileID    uuid.UUID `json:"profile_id"`
	SupplementID uuid.UUID `json:"supplement_id"`
	TimeMinutes  int       `json:"time_minutes"`
	DaysMask     int       `json:"days_mask"`
	IsEnabled    bool      `json:"is_enabled"`
}

type ReplaceScheduleItem struct {
	SupplementID uuid.UUID `json:"supplement_id"`
	TimeMinutes  int       `json:"time_minutes"`
	DaysMask     int       `json:"days_mask"`
	IsEnabled    bool      `json:"is_enabled"`
}

type ReplaceSchedulesRequest struct {
	ProfileID uuid.UUID             `json:"profile_id"`
	Schedules []ReplaceScheduleItem `json:"schedules"`
	Replace   bool                  `json:"replace"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (r UpsertScheduleRequest) Validate() error {
	if r.ProfileID == uuid.Nil {
		return fmt.Errorf("profile_id is required")
	}
	if r.SupplementID == uuid.Nil {
		return fmt.Errorf("supplement_id is required")
	}
	if r.TimeMinutes < 0 || r.TimeMinutes > 1439 {
		return fmt.Errorf("time_minutes must be in range 0..1439")
	}
	if r.DaysMask < 0 || r.DaysMask > 127 {
		return fmt.Errorf("days_mask must be in range 0..127")
	}
	return nil
}

func (r ReplaceSchedulesRequest) Validate() error {
	if r.ProfileID == uuid.Nil {
		return fmt.Errorf("profile_id is required")
	}
	if !r.Replace {
		return fmt.Errorf("replace must be true")
	}
	if len(r.Schedules) > maxSchedulesPerProfile {
		return fmt.Errorf("too many schedules")
	}
	for i, item := range r.Schedules {
		if item.SupplementID == uuid.Nil {
			return fmt.Errorf("schedules[%d].supplement_id is required", i)
		}
		if item.TimeMinutes < 0 || item.TimeMinutes > 1439 {
			return fmt.Errorf("schedules[%d].time_minutes must be in range 0..1439", i)
		}
		if item.DaysMask < 0 || item.DaysMask > 127 {
			return fmt.Errorf("schedules[%d].days_mask must be in range 0..127", i)
		}
	}
	return nil
}

func toDTO(row storage.SupplementSchedule) ScheduleDTO {
	return ScheduleDTO{
		ID:           row.ID,
		SupplementID: row.SupplementID,
		TimeMinutes:  row.TimeMinutes,
		DaysMask:     row.DaysMask,
		IsEnabled:    row.IsEnabled,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func normalizeOwner(userID string) string {
	return strings.TrimSpace(userID)
}
