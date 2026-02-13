package workouts

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// DTOs
// ============================================================================

// PlanDTO represents a workout plan for API responses.
type PlanDTO struct {
	ID        uuid.UUID `json:"id"`
	ProfileID uuid.UUID `json:"profile_id"`
	Title     string    `json:"title"`
	Goal      string    `json:"goal"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ItemDTO represents a workout plan item for API responses.
type ItemDTO struct {
	ID          uuid.UUID       `json:"id"`
	Kind        string          `json:"kind"`
	TimeMinutes int             `json:"time_minutes"`
	DaysMask    int             `json:"days_mask"`
	DurationMin int             `json:"duration_min"`
	Intensity   string          `json:"intensity"`
	Note        string          `json:"note"`
	Details     json.RawMessage `json:"details"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// CompletionDTO represents a workout completion record for API responses.
type CompletionDTO struct {
	ID         uuid.UUID `json:"id"`
	Date       string    `json:"date"`
	PlanItemID uuid.UUID `json:"plan_item_id"`
	Status     string    `json:"status"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// WorkoutDTO represents an actual workout session from feed/day.
type WorkoutDTO struct {
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	Label        string    `json:"label"`
	CaloriesKcal *int      `json:"calories_kcal,omitempty"`
}

// ============================================================================
// Requests
// ============================================================================

// UpsertPlanRequest is used to create or update a workout plan.
type UpsertPlanRequest struct {
	ProfileID uuid.UUID `json:"profile_id"`
	Title     string    `json:"title"`
	Goal      string    `json:"goal"`
}

// ItemUpsertRequest represents a single item in a plan replacement.
type ItemUpsertRequest struct {
	Kind        string          `json:"kind"`
	TimeMinutes int             `json:"time_minutes"`
	DaysMask    int             `json:"days_mask"`
	DurationMin int             `json:"duration_min"`
	Intensity   string          `json:"intensity"`
	Note        string          `json:"note"`
	Details     json.RawMessage `json:"details"`
}

// ReplaceItemsRequest is used to replace all items in a workout plan.
type ReplaceItemsRequest struct {
	ProfileID uuid.UUID           `json:"profile_id"`
	Title     string              `json:"title"`
	Goal      string              `json:"goal"`
	Replace   bool                `json:"replace"`
	Items     []ItemUpsertRequest `json:"items"`
}

// UpsertCompletionRequest is used to mark a workout as done or skipped.
type UpsertCompletionRequest struct {
	ProfileID  uuid.UUID `json:"profile_id"`
	Date       string    `json:"date"`
	PlanItemID uuid.UUID `json:"plan_item_id"`
	Status     string    `json:"status"`
	Note       string    `json:"note"`
}

// ============================================================================
// Responses
// ============================================================================

// GetPlanResponse returns a plan with its items.
type GetPlanResponse struct {
	Plan  *PlanDTO  `json:"plan"`
	Items []ItemDTO `json:"items"`
}

// ReplaceItemsResponse returns the updated plan and items.
type ReplaceItemsResponse struct {
	Plan  PlanDTO   `json:"plan"`
	Items []ItemDTO `json:"items"`
}

// TodayResponse returns today's workout plan and completion status.
type TodayResponse struct {
	Date           string          `json:"date"`
	ProfileID      uuid.UUID       `json:"profile_id"`
	Planned        []ItemDTO       `json:"planned"`
	Completions    []CompletionDTO `json:"completions"`
	ActualWorkouts []WorkoutDTO    `json:"actual_workouts"`
	IsDone         bool            `json:"is_done"`
}

// ListCompletionsResponse returns a list of completions.
type ListCompletionsResponse struct {
	Completions []CompletionDTO `json:"completions"`
}

// ============================================================================
// Validation
// ============================================================================

const (
	MaxItems       = 30
	MaxItemsPerDay = 4
	MaxDetailsSize = 16 * 1024 // 16KB
	MinDuration    = 5
	MaxDuration    = 240
	MinTimeMinutes = 0
	MaxTimeMinutes = 1439
	MinDaysMask    = 0
	MaxDaysMask    = 127
)

var (
	ValidKinds = map[string]bool{
		"run":      true,
		"walk":     true,
		"strength": true,
		"morning":  true,
		"core":     true,
		"other":    true,
	}

	ValidIntensities = map[string]bool{
		"low":    true,
		"medium": true,
		"high":   true,
	}

	ValidStatuses = map[string]bool{
		"done":    true,
		"skipped": true,
	}
)

// ValidateReplaceItemsRequest validates a replace items request.
func ValidateReplaceItemsRequest(req *ReplaceItemsRequest) error {
	if req.ProfileID == uuid.Nil {
		return fmt.Errorf("profile_id is required")
	}

	if !req.Replace {
		return fmt.Errorf("replace must be true")
	}

	if req.Title == "" {
		return fmt.Errorf("title is required")
	}

	if len(req.Items) == 0 {
		return fmt.Errorf("items cannot be empty")
	}

	if len(req.Items) > MaxItems {
		return fmt.Errorf("too many items: max %d", MaxItems)
	}

	// Track items per day
	dayItems := make(map[int]int) // day -> count

	for i, item := range req.Items {
		if err := ValidateItemUpsertRequest(&item); err != nil {
			return fmt.Errorf("item[%d]: %w", i, err)
		}

		// Count items per day
		for day := 0; day < 7; day++ {
			if item.DaysMask&(1<<day) != 0 {
				dayItems[day]++
				if dayItems[day] > MaxItemsPerDay {
					return fmt.Errorf("too many items on day %d: max %d per day", day, MaxItemsPerDay)
				}
			}
		}
	}

	return nil
}

// ValidateItemUpsertRequest validates a single item upsert request.
func ValidateItemUpsertRequest(item *ItemUpsertRequest) error {
	if !ValidKinds[item.Kind] {
		return fmt.Errorf("invalid kind: %s", item.Kind)
	}

	if item.TimeMinutes < MinTimeMinutes || item.TimeMinutes > MaxTimeMinutes {
		return fmt.Errorf("time_minutes must be between %d and %d", MinTimeMinutes, MaxTimeMinutes)
	}

	if item.DaysMask < MinDaysMask || item.DaysMask > MaxDaysMask {
		return fmt.Errorf("days_mask must be between %d and %d", MinDaysMask, MaxDaysMask)
	}

	if item.DurationMin < MinDuration || item.DurationMin > MaxDuration {
		return fmt.Errorf("duration_min must be between %d and %d", MinDuration, MaxDuration)
	}

	if item.Intensity == "" {
		item.Intensity = "medium"
	}

	if !ValidIntensities[item.Intensity] {
		return fmt.Errorf("invalid intensity: %s", item.Intensity)
	}

	// Validate details size
	if len(item.Details) > MaxDetailsSize {
		return fmt.Errorf("details too large: max %d bytes", MaxDetailsSize)
	}

	// Validate details is valid JSON (if not empty)
	if len(item.Details) > 0 {
		var test interface{}
		if err := json.Unmarshal(item.Details, &test); err != nil {
			return fmt.Errorf("details must be valid JSON: %w", err)
		}
	} else {
		item.Details = json.RawMessage("{}")
	}

	return nil
}

// ValidateUpsertCompletionRequest validates a completion upsert request.
func ValidateUpsertCompletionRequest(req *UpsertCompletionRequest) error {
	if req.ProfileID == uuid.Nil {
		return fmt.Errorf("profile_id is required")
	}

	if req.Date == "" {
		return fmt.Errorf("date is required")
	}

	// Validate date format YYYY-MM-DD
	_, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return fmt.Errorf("date must be in YYYY-MM-DD format")
	}

	if req.PlanItemID == uuid.Nil {
		return fmt.Errorf("plan_item_id is required")
	}

	if !ValidStatuses[req.Status] {
		return fmt.Errorf("invalid status: %s (must be done or skipped)", req.Status)
	}

	return nil
}
