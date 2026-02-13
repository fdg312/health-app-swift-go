package memory

import (
	"sync"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

// WorkoutCompletionsStorage implements storage.WorkoutCompletionsStorage in memory.
type WorkoutCompletionsStorage struct {
	mu          sync.RWMutex
	completions map[uuid.UUID]storage.WorkoutCompletion // id -> completion
	// index: ownerUserID+profileID+date+planItemID -> completionID
	uniqueIndex map[string]uuid.UUID
}

func NewWorkoutCompletionsStorage() *WorkoutCompletionsStorage {
	return &WorkoutCompletionsStorage{
		completions: make(map[uuid.UUID]storage.WorkoutCompletion),
		uniqueIndex: make(map[string]uuid.UUID),
	}
}

func (s *WorkoutCompletionsStorage) UpsertCompletion(ownerUserID string, profileID uuid.UUID, date string, planItemID uuid.UUID, status string, note string) (storage.WorkoutCompletion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := ownerUserID + ":" + profileID.String() + ":" + date + ":" + planItemID.String()
	now := time.Now()

	// Check if exists
	if existingID, ok := s.uniqueIndex[key]; ok {
		// Update existing
		completion := s.completions[existingID]
		completion.Status = status
		completion.Note = note
		completion.UpdatedAt = now
		s.completions[existingID] = completion
		return completion, nil
	}

	// Create new
	completion := storage.WorkoutCompletion{
		ID:          uuid.New(),
		OwnerUserID: ownerUserID,
		ProfileID:   profileID,
		Date:        date,
		PlanItemID:  planItemID,
		Status:      status,
		Note:        note,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.completions[completion.ID] = completion
	s.uniqueIndex[key] = completion.ID

	return completion, nil
}

func (s *WorkoutCompletionsStorage) ListCompletions(ownerUserID string, profileID uuid.UUID, from string, to string) ([]storage.WorkoutCompletion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []storage.WorkoutCompletion
	for _, completion := range s.completions {
		// Verify ownership
		if completion.OwnerUserID != ownerUserID || completion.ProfileID != profileID {
			continue
		}

		// Date range filter
		if completion.Date < from || completion.Date > to {
			continue
		}

		result = append(result, completion)
	}

	return result, nil
}
