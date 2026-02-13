package foodprefs

import (
	"fmt"
	"time"
)

// FoodPrefDTO represents a user-defined food item with nutritional information.
type FoodPrefDTO struct {
	ID              string    `json:"id"`
	ProfileID       string    `json:"profile_id"`
	Name            string    `json:"name"`
	Tags            []string  `json:"tags"`
	KcalPer100g     int       `json:"kcal_per_100g"`
	ProteinGPer100g int       `json:"protein_g_per_100g"`
	FatGPer100g     int       `json:"fat_g_per_100g"`
	CarbsGPer100g   int       `json:"carbs_g_per_100g"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ListFoodPrefsResponse is the response for GET /v1/food/prefs.
type ListFoodPrefsResponse struct {
	Items  []FoodPrefDTO `json:"items"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

// UpsertFoodPrefRequest is the request body for POST /v1/food/prefs.
type UpsertFoodPrefRequest struct {
	ID              string   `json:"id,omitempty"` // if provided, update existing
	ProfileID       string   `json:"profile_id"`
	Name            string   `json:"name"`
	Tags            []string `json:"tags"`
	KcalPer100g     int      `json:"kcal_per_100g"`
	ProteinGPer100g int      `json:"protein_g_per_100g"`
	FatGPer100g     int      `json:"fat_g_per_100g"`
	CarbsGPer100g   int      `json:"carbs_g_per_100g"`
}

// Validate validates the upsert request.
func (r *UpsertFoodPrefRequest) Validate() error {
	if r.ProfileID == "" {
		return fmt.Errorf("profile_id is required")
	}

	if len(r.Name) < 1 || len(r.Name) > 80 {
		return fmt.Errorf("name must be between 1 and 80 characters")
	}

	if r.KcalPer100g < 0 || r.KcalPer100g > 1000 {
		return fmt.Errorf("kcal_per_100g must be between 0 and 1000")
	}

	if r.ProteinGPer100g < 0 || r.ProteinGPer100g > 1000 {
		return fmt.Errorf("protein_g_per_100g must be between 0 and 1000")
	}

	if r.FatGPer100g < 0 || r.FatGPer100g > 1000 {
		return fmt.Errorf("fat_g_per_100g must be between 0 and 1000")
	}

	if r.CarbsGPer100g < 0 || r.CarbsGPer100g > 1000 {
		return fmt.Errorf("carbs_g_per_100g must be between 0 and 1000")
	}

	if r.Tags == nil {
		r.Tags = []string{}
	}

	return nil
}
