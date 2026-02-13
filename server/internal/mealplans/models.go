package mealplans

import (
	"fmt"
	"time"
)

type MealPlanDTO struct {
	ID        string     `json:"id"`
	ProfileID string     `json:"profile_id"`
	Title     string     `json:"title"`
	IsActive  bool       `json:"is_active"`
	FromDate  *time.Time `json:"from_date,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type MealPlanItemDTO struct {
	ID             string    `json:"id"`
	ProfileID      string    `json:"profile_id"`
	PlanID         string    `json:"plan_id"`
	DayIndex       int       `json:"day_index"`
	MealSlot       string    `json:"meal_slot"`
	Title          string    `json:"title"`
	Notes          string    `json:"notes"`
	ApproxKcal     int       `json:"approx_kcal"`
	ApproxProteinG int       `json:"approx_protein_g"`
	ApproxFatG     int       `json:"approx_fat_g"`
	ApproxCarbsG   int       `json:"approx_carbs_g"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type GetMealPlanResponse struct {
	Plan  *MealPlanDTO      `json:"plan"`
	Items []MealPlanItemDTO `json:"items"`
}

type ReplaceMealPlanRequest struct {
	ProfileID string                    `json:"profile_id"`
	Title     string                    `json:"title"`
	Items     []MealPlanItemUpsertInput `json:"items"`
}

type MealPlanItemUpsertInput struct {
	DayIndex       int    `json:"day_index"`
	MealSlot       string `json:"meal_slot"`
	Title          string `json:"title"`
	Notes          string `json:"notes"`
	ApproxKcal     int    `json:"approx_kcal"`
	ApproxProteinG int    `json:"approx_protein_g"`
	ApproxFatG     int    `json:"approx_fat_g"`
	ApproxCarbsG   int    `json:"approx_carbs_g"`
}

type GetTodayResponse struct {
	Date  string            `json:"date"`
	Items []MealPlanItemDTO `json:"items"`
}

func (r *ReplaceMealPlanRequest) Validate() error {
	if r.ProfileID == "" {
		return fmt.Errorf("profile_id is required")
	}
	if len(r.Title) < 1 || len(r.Title) > 200 {
		return fmt.Errorf("title must be between 1 and 200 characters")
	}
	if len(r.Items) == 0 {
		return fmt.Errorf("items is required and must not be empty")
	}
	if len(r.Items) > 28 {
		return fmt.Errorf("items cannot exceed 28")
	}
	seen := make(map[string]bool)
	validSlots := map[string]bool{"breakfast": true, "lunch": true, "dinner": true, "snack": true}
	for i, item := range r.Items {
		if item.DayIndex < 0 || item.DayIndex > 6 {
			return fmt.Errorf("item[%d]: day_index must be 0-6", i)
		}
		if !validSlots[item.MealSlot] {
			return fmt.Errorf("item[%d]: invalid meal_slot", i)
		}
		if len(item.Title) < 1 || len(item.Title) > 200 {
			return fmt.Errorf("item[%d]: title must be 1-200 chars", i)
		}
		key := fmt.Sprintf("%d:%s", item.DayIndex, item.MealSlot)
		if seen[key] {
			return fmt.Errorf("duplicate (day_index, meal_slot): %s", key)
		}
		seen[key] = true
		if item.ApproxKcal < 0 || item.ApproxKcal > 10000 {
			return fmt.Errorf("item[%d]: approx_kcal must be 0-10000", i)
		}
		if item.ApproxProteinG < 0 || item.ApproxProteinG > 1000 {
			return fmt.Errorf("item[%d]: approx_protein_g must be 0-1000", i)
		}
		if item.ApproxFatG < 0 || item.ApproxFatG > 1000 {
			return fmt.Errorf("item[%d]: approx_fat_g must be 0-1000", i)
		}
		if item.ApproxCarbsG < 0 || item.ApproxCarbsG > 1000 {
			return fmt.Errorf("item[%d]: approx_carbs_g must be 0-1000", i)
		}
	}
	return nil
}
