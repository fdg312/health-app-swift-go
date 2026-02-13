package proposals

import (
	"encoding/json"
	"time"

	"github.com/fdg312/health-hub/internal/settings"
	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

type ProposalDTO struct {
	ID        uuid.UUID      `json:"id"`
	ProfileID uuid.UUID      `json:"profile_id"`
	Status    string         `json:"status"`
	Kind      string         `json:"kind"`
	Title     string         `json:"title"`
	Summary   string         `json:"summary"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}

type ListProposalsResponse struct {
	Proposals []ProposalDTO `json:"proposals"`
}

type ApplyProposalResponse struct {
	Status  string            `json:"status"`
	Applied *AppliedResultDTO `json:"applied,omitempty"`
}

type AppliedResultDTO struct {
	Settings             *settings.SettingsDTO `json:"settings,omitempty"`
	SchedulesCreated     *int                  `json:"schedules_created,omitempty"`
	WorkoutItemsCreated  *int                  `json:"workout_items_created,omitempty"`
	NutritionTargets     *bool                 `json:"nutrition_targets_updated,omitempty"`
	MealPlanItemsCreated *int                  `json:"meal_plan_items_created,omitempty"`
}

type RejectProposalResponse struct {
	Status string `json:"status"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func proposalToDTO(p storage.AIProposal) ProposalDTO {
	payload := make(map[string]any)
	if len(p.Payload) > 0 {
		_ = json.Unmarshal(p.Payload, &payload)
	}
	return ProposalDTO{
		ID:        p.ID,
		ProfileID: p.ProfileID,
		Status:    p.Status,
		Kind:      p.Kind,
		Title:     p.Title,
		Summary:   p.Summary,
		Payload:   payload,
		CreatedAt: p.CreatedAt,
	}
}

// WorkoutPlanPayload represents the payload for workout_plan proposals
type WorkoutPlanPayload struct {
	Replace bool                     `json:"replace"`
	Title   string                   `json:"title"`
	Goal    string                   `json:"goal"`
	Items   []WorkoutPlanPayloadItem `json:"items"`
}

// WorkoutPlanPayloadItem represents a single workout item in the payload
type WorkoutPlanPayloadItem struct {
	Kind        string          `json:"kind"`
	TimeMinutes int             `json:"time_minutes"`
	DaysMask    int             `json:"days_mask"`
	DurationMin int             `json:"duration_min"`
	Intensity   string          `json:"intensity"`
	Note        string          `json:"note"`
	Details     json.RawMessage `json:"details"`
}

func parseWorkoutPlanPayload(data []byte) (*WorkoutPlanPayload, error) {
	var payload WorkoutPlanPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// NutritionPlanPayload represents the payload for nutrition_plan proposals
type NutritionPlanPayload struct {
	CaloriesKcal int `json:"calories_kcal"`
	ProteinG     int `json:"protein_g"`
	FatG         int `json:"fat_g"`
	CarbsG       int `json:"carbs_g"`
	CalciumMg    int `json:"calcium_mg"`
}

func parseNutritionPlanPayload(data []byte) (*NutritionPlanPayload, error) {
	var payload NutritionPlanPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// MealPlanPayload represents the payload for meal_plan proposals
type MealPlanPayload struct {
	Title string                `json:"title"`
	Items []MealPlanPayloadItem `json:"items"`
}

// MealPlanPayloadItem represents a single meal item in the payload
type MealPlanPayloadItem struct {
	DayIndex       int    `json:"day_index"`
	MealSlot       string `json:"meal_slot"`
	Title          string `json:"title"`
	Notes          string `json:"notes"`
	ApproxKcal     int    `json:"approx_kcal"`
	ApproxProteinG int    `json:"approx_protein_g"`
	ApproxFatG     int    `json:"approx_fat_g"`
	ApproxCarbsG   int    `json:"approx_carbs_g"`
}

func parseMealPlanPayload(data []byte) (*MealPlanPayload, error) {
	var payload MealPlanPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}
