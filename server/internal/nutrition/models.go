package nutrition

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TargetsDTO represents nutrition targets/goals for a profile.
type TargetsDTO struct {
	ProfileID    uuid.UUID `json:"profile_id"`
	CaloriesKcal int       `json:"calories_kcal"`
	ProteinG     int       `json:"protein_g"`
	FatG         int       `json:"fat_g"`
	CarbsG       int       `json:"carbs_g"`
	CalciumMg    int       `json:"calcium_mg"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// GetTargetsResponse contains targets and a flag indicating if they are defaults.
type GetTargetsResponse struct {
	Targets   TargetsDTO `json:"targets"`
	IsDefault bool       `json:"is_default"`
}

// UpsertTargetsRequest is the request body for PUT /v1/nutrition/targets.
type UpsertTargetsRequest struct {
	ProfileID    uuid.UUID `json:"profile_id"`
	CaloriesKcal int       `json:"calories_kcal"`
	ProteinG     int       `json:"protein_g"`
	FatG         int       `json:"fat_g"`
	CarbsG       int       `json:"carbs_g"`
	CalciumMg    int       `json:"calcium_mg"`
}

// Validate validates the upsert request.
func (r *UpsertTargetsRequest) Validate() error {
	if r.ProfileID == uuid.Nil {
		return fmt.Errorf("profile_id is required")
	}

	if r.CaloriesKcal < 800 || r.CaloriesKcal > 6000 {
		return fmt.Errorf("calories_kcal must be between 800 and 6000")
	}

	if r.ProteinG < 0 || r.ProteinG > 400 {
		return fmt.Errorf("protein_g must be between 0 and 400")
	}

	if r.FatG < 0 || r.FatG > 400 {
		return fmt.Errorf("fat_g must be between 0 and 400")
	}

	if r.CarbsG < 0 || r.CarbsG > 400 {
		return fmt.Errorf("carbs_g must be between 0 and 400")
	}

	if r.CalciumMg < 0 || r.CalciumMg > 5000 {
		return fmt.Errorf("calcium_mg must be between 0 and 5000")
	}

	return nil
}

// GetDefaultTargets returns reasonable default nutrition targets.
func GetDefaultTargets(profileID uuid.UUID) TargetsDTO {
	now := time.Now().UTC()
	return TargetsDTO{
		ProfileID:    profileID,
		CaloriesKcal: 2200,
		ProteinG:     120,
		FatG:         70,
		CarbsG:       250,
		CalciumMg:    800,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
