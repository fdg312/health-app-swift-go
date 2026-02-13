package memory

import (
	"errors"
	"sync"

	"github.com/fdg312/health-hub/internal/checkins"
	"github.com/google/uuid"
)

// CheckinsMemoryStorage implements checkins.Storage
type CheckinsMemoryStorage struct {
	mu       sync.RWMutex
	checkins map[uuid.UUID]checkins.Checkin          // by ID
	byKey    map[string]uuid.UUID                     // key: "profileID:date:type" -> checkin ID
}

// NewCheckinsMemoryStorage creates a new in-memory checkins storage
func NewCheckinsMemoryStorage() *CheckinsMemoryStorage {
	return &CheckinsMemoryStorage{
		checkins: make(map[uuid.UUID]checkins.Checkin),
		byKey:    make(map[string]uuid.UUID),
	}
}

// ListCheckins returns all check-ins for a profile within a date range
func (s *CheckinsMemoryStorage) ListCheckins(profileID uuid.UUID, from, to string) ([]checkins.Checkin, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []checkins.Checkin
	for _, c := range s.checkins {
		if c.ProfileID == profileID && c.Date >= from && c.Date <= to {
			result = append(result, c)
		}
	}

	return result, nil
}

// GetCheckin retrieves a check-in by ID
func (s *CheckinsMemoryStorage) GetCheckin(id uuid.UUID) (*checkins.Checkin, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c, exists := s.checkins[id]
	if !exists {
		return nil, errors.New("checkin not found")
	}

	return &c, nil
}

// UpsertCheckin creates or updates a check-in (by profile_id, date, type)
func (s *CheckinsMemoryStorage) UpsertCheckin(checkin *checkins.Checkin) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := makeKey(checkin.ProfileID, checkin.Date, checkin.Type)

	// Check if checkin with same key exists
	if existingID, exists := s.byKey[key]; exists {
		// Update existing checkin
		existing := s.checkins[existingID]
		existing.Score = checkin.Score
		existing.Tags = checkin.Tags
		existing.Note = checkin.Note
		existing.UpdatedAt = checkin.UpdatedAt
		s.checkins[existingID] = existing

		// Update the checkin ID to return
		*checkin = existing
	} else {
		// Create new checkin
		s.checkins[checkin.ID] = *checkin
		s.byKey[key] = checkin.ID
	}

	return nil
}

// DeleteCheckin deletes a check-in by ID
func (s *CheckinsMemoryStorage) DeleteCheckin(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, exists := s.checkins[id]
	if !exists {
		return errors.New("checkin not found")
	}

	key := makeKey(c.ProfileID, c.Date, c.Type)
	delete(s.checkins, id)
	delete(s.byKey, key)

	return nil
}

func makeKey(profileID uuid.UUID, date, ctype string) string {
	return profileID.String() + ":" + date + ":" + ctype
}
