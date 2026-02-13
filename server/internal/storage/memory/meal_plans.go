package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

type mealPlansStorage struct {
	mu    sync.RWMutex
	plans map[string]*storage.MealPlan     // key: plan_id
	items map[string]*storage.MealPlanItem // key: item_id
	// index for owner+profile lookups
	byOwnerProfile map[string]string   // key: "ownerUserID:profileID" -> active plan_id
	itemsByPlan    map[string][]string // key: plan_id -> []item_id
}

func newMealPlansStorage() *mealPlansStorage {
	return &mealPlansStorage{
		plans:          make(map[string]*storage.MealPlan),
		items:          make(map[string]*storage.MealPlanItem),
		byOwnerProfile: make(map[string]string),
		itemsByPlan:    make(map[string][]string),
	}
}

func (s *mealPlansStorage) GetActive(ctx context.Context, ownerUserID string, profileID string) (storage.MealPlan, []storage.MealPlanItem, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", ownerUserID, profileID)
	planID, ok := s.byOwnerProfile[key]
	if !ok {
		return storage.MealPlan{}, nil, false, nil
	}

	plan, ok := s.plans[planID]
	if !ok {
		return storage.MealPlan{}, nil, false, nil
	}

	// Get items for this plan
	items := s.getItemsByPlanIDLocked(planID)

	return *plan, items, true, nil
}

func (s *mealPlansStorage) ReplaceActive(ctx context.Context, ownerUserID string, profileID string, title string, itemsUpsert []storage.MealPlanItemUpsert) (storage.MealPlan, []storage.MealPlanItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	key := fmt.Sprintf("%s:%s", ownerUserID, profileID)

	// Delete existing active plan if any
	if existingPlanID, ok := s.byOwnerProfile[key]; ok {
		s.deletePlanAndItemsLocked(existingPlanID)
	}

	// Create new plan
	newPlanID := uuid.New().String()
	plan := &storage.MealPlan{
		ID:          newPlanID,
		OwnerUserID: ownerUserID,
		ProfileID:   profileID,
		Title:       title,
		IsActive:    true,
		FromDate:    nil,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.plans[newPlanID] = plan
	s.byOwnerProfile[key] = newPlanID

	// Create items
	var items []storage.MealPlanItem
	for _, itemReq := range itemsUpsert {
		itemID := uuid.New().String()
		item := &storage.MealPlanItem{
			ID:             itemID,
			OwnerUserID:    ownerUserID,
			ProfileID:      profileID,
			PlanID:         newPlanID,
			DayIndex:       itemReq.DayIndex,
			MealSlot:       itemReq.MealSlot,
			Title:          itemReq.Title,
			Notes:          itemReq.Notes,
			ApproxKcal:     itemReq.ApproxKcal,
			ApproxProteinG: itemReq.ApproxProteinG,
			ApproxFatG:     itemReq.ApproxFatG,
			ApproxCarbsG:   itemReq.ApproxCarbsG,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		s.items[itemID] = item
		s.itemsByPlan[newPlanID] = append(s.itemsByPlan[newPlanID], itemID)
		items = append(items, *item)
	}

	return *plan, items, nil
}

func (s *mealPlansStorage) DeleteActive(ctx context.Context, ownerUserID string, profileID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s", ownerUserID, profileID)
	planID, ok := s.byOwnerProfile[key]
	if !ok {
		return nil // nothing to delete
	}

	// Verify ownership
	plan, ok := s.plans[planID]
	if !ok || plan.OwnerUserID != ownerUserID {
		return fmt.Errorf("unauthorized or plan not found")
	}

	s.deletePlanAndItemsLocked(planID)
	delete(s.byOwnerProfile, key)

	return nil
}

func (s *mealPlansStorage) GetToday(ctx context.Context, ownerUserID string, profileID string, date time.Time) ([]storage.MealPlanItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", ownerUserID, profileID)
	planID, ok := s.byOwnerProfile[key]
	if !ok {
		return []storage.MealPlanItem{}, nil
	}

	// Calculate day_index from date (0=Monday, 6=Sunday)
	// In Go, Sunday=0, Monday=1, ..., Saturday=6
	weekday := date.Weekday()
	dayIndex := int(weekday) - 1
	if dayIndex < 0 {
		dayIndex = 6 // Sunday becomes 6
	}

	// Get items for this plan and day
	itemIDs, ok := s.itemsByPlan[planID]
	if !ok {
		return []storage.MealPlanItem{}, nil
	}

	var results []storage.MealPlanItem
	for _, itemID := range itemIDs {
		item, ok := s.items[itemID]
		if !ok {
			continue
		}

		if item.DayIndex == dayIndex {
			results = append(results, *item)
		}
	}

	return results, nil
}

// Helper methods (must be called with lock held)
func (s *mealPlansStorage) getItemsByPlanIDLocked(planID string) []storage.MealPlanItem {
	itemIDs, ok := s.itemsByPlan[planID]
	if !ok {
		return []storage.MealPlanItem{}
	}

	var items []storage.MealPlanItem
	for _, itemID := range itemIDs {
		if item, ok := s.items[itemID]; ok {
			items = append(items, *item)
		}
	}

	return items
}

func (s *mealPlansStorage) deletePlanAndItemsLocked(planID string) {
	// Delete all items
	if itemIDs, ok := s.itemsByPlan[planID]; ok {
		for _, itemID := range itemIDs {
			delete(s.items, itemID)
		}
		delete(s.itemsByPlan, planID)
	}

	// Delete plan
	delete(s.plans, planID)
}
