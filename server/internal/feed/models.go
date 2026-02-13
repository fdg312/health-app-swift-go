package feed

import (
	"time"

	"github.com/google/uuid"
)

// FeedDayResponse is the response for GET /v1/feed/day
type FeedDayResponse struct {
	Date              string             `json:"date"`
	ProfileID         uuid.UUID          `json:"profile_id"`
	Daily             interface{}        `json:"daily"` // DailyAggregate from metrics package (as JSON)
	Checkins          *DayCheckins       `json:"checkins"`
	Intakes           *IntakesSummary    `json:"intakes,omitempty"`
	NutritionTargets  *NutritionTargets  `json:"nutrition_targets,omitempty"`
	NutritionProgress *NutritionProgress `json:"nutrition_progress,omitempty"`
	MealToday         []MealPlanItem     `json:"meal_today,omitempty"`
	MealPlanTitle     string             `json:"meal_plan_title,omitempty"`
	FoodPrefsCount    int                `json:"food_prefs_count"`
	MissingFields     []string           `json:"missing_fields"` // e.g., ["daily", "morning_checkin", "evening_checkin", "weight", "resting_hr"]
}

// IntakesSummary contains water and supplements summary for the day
type IntakesSummary struct {
	WaterTotalMl     int `json:"water_total_ml"`
	SupplementsTaken int `json:"supplements_taken"`
	SupplementsTotal int `json:"supplements_total"`
}

// NutritionTargets represents nutrition targets/goals
type NutritionTargets struct {
	CaloriesKcal int `json:"calories_kcal"`
	ProteinG     int `json:"protein_g"`
	FatG         int `json:"fat_g"`
	CarbsG       int `json:"carbs_g"`
	CalciumMg    int `json:"calcium_mg"`
}

// NutritionProgress represents actual nutrition intake vs targets
type NutritionProgress struct {
	CaloriesKcal    int `json:"calories_kcal"`
	CaloriesPercent int `json:"calories_percent"`
	ProteinG        int `json:"protein_g"`
	ProteinPercent  int `json:"protein_percent"`
	FatG            int `json:"fat_g"`
	FatPercent      int `json:"fat_percent"`
	CarbsG          int `json:"carbs_g"`
	CarbsPercent    int `json:"carbs_percent"`
	CalciumMg       int `json:"calcium_mg"`
	CalciumPercent  int `json:"calcium_percent"`
}

// DayCheckins contains morning and evening checkins
type DayCheckins struct {
	Morning *CheckinSummary `json:"morning,omitempty"`
	Evening *CheckinSummary `json:"evening,omitempty"`
}

// CheckinSummary is a simplified checkin for feed
type CheckinSummary struct {
	ID        uuid.UUID `json:"id"`
	Score     int       `json:"score"`
	Tags      []string  `json:"tags,omitempty"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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

// MealPlanItem represents a meal in the daily meal plan
type MealPlanItem struct {
	ID             uuid.UUID `json:"id"`
	DayIndex       int       `json:"day_index"`
	MealSlot       string    `json:"meal_slot"`
	Title          string    `json:"title"`
	Notes          string    `json:"notes"`
	ApproxKcal     int       `json:"approx_kcal"`
	ApproxProteinG int       `json:"approx_protein_g"`
	ApproxFatG     int       `json:"approx_fat_g"`
	ApproxCarbsG   int       `json:"approx_carbs_g"`
}
