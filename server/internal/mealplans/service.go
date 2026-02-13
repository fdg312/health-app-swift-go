package mealplans

import (
	"context"
	"fmt"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
)

// Service handles meal plans business logic.
type Service struct {
	storage storage.MealPlansStorage
}

// NewService creates a new meal plans service.
func NewService(storage storage.MealPlansStorage) *Service {
	return &Service{storage: storage}
}

// GetActive returns the active meal plan for a profile.
func (s *Service) GetActive(ctx context.Context, ownerUserID string, profileID string) (*MealPlanDTO, []MealPlanItemDTO, bool, error) {
	plan, items, found, err := s.storage.GetActive(ctx, ownerUserID, profileID)
	if err != nil {
		return nil, nil, false, err
	}
	if !found {
		return nil, nil, false, nil
	}

	planDTO := &MealPlanDTO{
		ID:        plan.ID,
		ProfileID: plan.ProfileID,
		Title:     plan.Title,
		IsActive:  plan.IsActive,
		FromDate:  plan.FromDate,
		CreatedAt: plan.CreatedAt,
		UpdatedAt: plan.UpdatedAt,
	}

	itemDTOs := make([]MealPlanItemDTO, len(items))
	for i, item := range items {
		itemDTOs[i] = toItemDTO(item)
	}

	return planDTO, itemDTOs, true, nil
}

// ReplaceActive replaces the active meal plan with new title and items.
func (s *Service) ReplaceActive(ctx context.Context, ownerUserID string, req ReplaceMealPlanRequest) (*MealPlanDTO, []MealPlanItemDTO, error) {
	if err := req.Validate(); err != nil {
		return nil, nil, fmt.Errorf("validation failed: %w", err)
	}

	// Convert request items to storage upserts
	items := make([]storage.MealPlanItemUpsert, len(req.Items))
	for i, item := range req.Items {
		items[i] = storage.MealPlanItemUpsert{
			DayIndex:       item.DayIndex,
			MealSlot:       item.MealSlot,
			Title:          item.Title,
			Notes:          item.Notes,
			ApproxKcal:     item.ApproxKcal,
			ApproxProteinG: item.ApproxProteinG,
			ApproxFatG:     item.ApproxFatG,
			ApproxCarbsG:   item.ApproxCarbsG,
		}
	}

	plan, createdItems, err := s.storage.ReplaceActive(ctx, ownerUserID, req.ProfileID, req.Title, items)
	if err != nil {
		return nil, nil, err
	}

	planDTO := &MealPlanDTO{
		ID:        plan.ID,
		ProfileID: plan.ProfileID,
		Title:     plan.Title,
		IsActive:  plan.IsActive,
		FromDate:  plan.FromDate,
		CreatedAt: plan.CreatedAt,
		UpdatedAt: plan.UpdatedAt,
	}

	itemDTOs := make([]MealPlanItemDTO, len(createdItems))
	for i, item := range createdItems {
		itemDTOs[i] = toItemDTO(item)
	}

	return planDTO, itemDTOs, nil
}

// DeleteActive deletes the active meal plan for a profile.
func (s *Service) DeleteActive(ctx context.Context, ownerUserID string, profileID string) error {
	return s.storage.DeleteActive(ctx, ownerUserID, profileID)
}

// GetToday returns meal plan items for a specific date.
func (s *Service) GetToday(ctx context.Context, ownerUserID string, profileID string, dateStr string) ([]MealPlanItemDTO, error) {
	var date time.Time
	var err error

	if dateStr == "" {
		date = time.Now().UTC()
	} else {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD")
		}
	}

	items, err := s.storage.GetToday(ctx, ownerUserID, profileID, date)
	if err != nil {
		return nil, err
	}

	itemDTOs := make([]MealPlanItemDTO, len(items))
	for i, item := range items {
		itemDTOs[i] = toItemDTO(item)
	}

	return itemDTOs, nil
}

func toItemDTO(item storage.MealPlanItem) MealPlanItemDTO {
	return MealPlanItemDTO{
		ID:             item.ID,
		ProfileID:      item.ProfileID,
		PlanID:         item.PlanID,
		DayIndex:       item.DayIndex,
		MealSlot:       item.MealSlot,
		Title:          item.Title,
		Notes:          item.Notes,
		ApproxKcal:     item.ApproxKcal,
		ApproxProteinG: item.ApproxProteinG,
		ApproxFatG:     item.ApproxFatG,
		ApproxCarbsG:   item.ApproxCarbsG,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}
