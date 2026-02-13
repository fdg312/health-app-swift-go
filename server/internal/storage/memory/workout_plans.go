package memory

import (
	"sync"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

// WorkoutPlansStorage implements storage.WorkoutPlansStorage in memory.
type WorkoutPlansStorage struct {
	mu    sync.RWMutex
	plans map[uuid.UUID]storage.WorkoutPlan // id -> plan
	// index: ownerUserID+profileID -> planID (active only)
	activeIndex map[string]uuid.UUID
}

func NewWorkoutPlansStorage() *WorkoutPlansStorage {
	return &WorkoutPlansStorage{
		plans:       make(map[uuid.UUID]storage.WorkoutPlan),
		activeIndex: make(map[string]uuid.UUID),
	}
}

func (s *WorkoutPlansStorage) GetActivePlan(ownerUserID string, profileID uuid.UUID) (storage.WorkoutPlan, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := ownerUserID + ":" + profileID.String()
	planID, ok := s.activeIndex[key]
	if !ok {
		return storage.WorkoutPlan{}, false, nil
	}

	plan, ok := s.plans[planID]
	if !ok {
		return storage.WorkoutPlan{}, false, nil
	}

	// Verify ownership
	if plan.OwnerUserID != ownerUserID || plan.ProfileID != profileID {
		return storage.WorkoutPlan{}, false, nil
	}

	return plan, true, nil
}

func (s *WorkoutPlansStorage) UpsertActivePlan(ownerUserID string, profileID uuid.UUID, title string, goal string) (storage.WorkoutPlan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := ownerUserID + ":" + profileID.String()
	now := time.Now()

	// Deactivate existing active plan
	if existingID, ok := s.activeIndex[key]; ok {
		if existing, found := s.plans[existingID]; found {
			existing.IsActive = false
			existing.UpdatedAt = now
			s.plans[existingID] = existing
		}
	}

	// Create new active plan
	plan := storage.WorkoutPlan{
		ID:          uuid.New(),
		OwnerUserID: ownerUserID,
		ProfileID:   profileID,
		Title:       title,
		Goal:        goal,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.plans[plan.ID] = plan
	s.activeIndex[key] = plan.ID

	return plan, nil
}
