package memory

import (
	"sync"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

// WorkoutPlanItemsStorage implements storage.WorkoutPlanItemsStorage in memory.
type WorkoutPlanItemsStorage struct {
	mu    sync.RWMutex
	items map[uuid.UUID]storage.WorkoutPlanItem // id -> item
	// index: planID -> []itemID
	planIndex map[uuid.UUID][]uuid.UUID
}

func NewWorkoutPlanItemsStorage() *WorkoutPlanItemsStorage {
	return &WorkoutPlanItemsStorage{
		items:     make(map[uuid.UUID]storage.WorkoutPlanItem),
		planIndex: make(map[uuid.UUID][]uuid.UUID),
	}
}

func (s *WorkoutPlanItemsStorage) ListItems(ownerUserID string, profileID uuid.UUID, planID uuid.UUID) ([]storage.WorkoutPlanItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	itemIDs, ok := s.planIndex[planID]
	if !ok {
		return []storage.WorkoutPlanItem{}, nil
	}

	var result []storage.WorkoutPlanItem
	for _, itemID := range itemIDs {
		item, ok := s.items[itemID]
		if !ok {
			continue
		}
		// Verify ownership
		if item.OwnerUserID != ownerUserID || item.ProfileID != profileID {
			continue
		}
		result = append(result, item)
	}

	return result, nil
}

func (s *WorkoutPlanItemsStorage) ReplaceAllItems(ownerUserID string, profileID uuid.UUID, planID uuid.UUID, items []storage.WorkoutItemUpsert) ([]storage.WorkoutPlanItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Delete existing items for this plan
	if itemIDs, ok := s.planIndex[planID]; ok {
		for _, itemID := range itemIDs {
			delete(s.items, itemID)
		}
	}

	// Create new items
	now := time.Now()
	var newItems []storage.WorkoutPlanItem
	var newItemIDs []uuid.UUID

	for _, upsert := range items {
		item := storage.WorkoutPlanItem{
			ID:          uuid.New(),
			PlanID:      planID,
			OwnerUserID: ownerUserID,
			ProfileID:   profileID,
			Kind:        upsert.Kind,
			TimeMinutes: upsert.TimeMinutes,
			DaysMask:    upsert.DaysMask,
			DurationMin: upsert.DurationMin,
			Intensity:   upsert.Intensity,
			Note:        upsert.Note,
			Details:     upsert.Details,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		s.items[item.ID] = item
		newItems = append(newItems, item)
		newItemIDs = append(newItemIDs, item.ID)
	}

	s.planIndex[planID] = newItemIDs

	return newItems, nil
}

func (s *WorkoutPlanItemsStorage) DeleteItem(ownerUserID string, itemID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[itemID]
	if !ok {
		return nil // already deleted
	}

	// Verify ownership
	if item.OwnerUserID != ownerUserID {
		return nil // not found for this user
	}

	// Remove from items
	delete(s.items, itemID)

	// Remove from plan index
	if itemIDs, ok := s.planIndex[item.PlanID]; ok {
		var filtered []uuid.UUID
		for _, id := range itemIDs {
			if id != itemID {
				filtered = append(filtered, id)
			}
		}
		s.planIndex[item.PlanID] = filtered
	}

	return nil
}
