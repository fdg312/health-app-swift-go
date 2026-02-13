package memory

import (
	"context"
	"sync"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

// SupplementsMemoryStorage — in-memory storage for supplements
type SupplementsMemoryStorage struct {
	mu           sync.RWMutex
	supplements  map[uuid.UUID]*storage.Supplement
	components   map[uuid.UUID][]storage.SupplementComponent // supplement_id -> components
	byProfile    map[uuid.UUID][]uuid.UUID                   // profile_id -> supplement_ids
}

func NewSupplementsMemoryStorage() *SupplementsMemoryStorage {
	return &SupplementsMemoryStorage{
		supplements: make(map[uuid.UUID]*storage.Supplement),
		components:  make(map[uuid.UUID][]storage.SupplementComponent),
		byProfile:   make(map[uuid.UUID][]uuid.UUID),
	}
}

func (s *SupplementsMemoryStorage) CreateSupplement(ctx context.Context, supplement *storage.Supplement) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if supplement.ID == uuid.Nil {
		supplement.ID = uuid.New()
	}
	if supplement.CreatedAt.IsZero() {
		supplement.CreatedAt = time.Now()
		supplement.UpdatedAt = time.Now()
	}

	clone := *supplement
	s.supplements[clone.ID] = &clone
	s.byProfile[clone.ProfileID] = append(s.byProfile[clone.ProfileID], clone.ID)

	return nil
}

func (s *SupplementsMemoryStorage) GetSupplement(ctx context.Context, id uuid.UUID) (*storage.Supplement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if supplement, ok := s.supplements[id]; ok {
		clone := *supplement
		return &clone, nil
	}

	return nil, ErrNotFound
}

func (s *SupplementsMemoryStorage) ListSupplements(ctx context.Context, profileID uuid.UUID) ([]storage.Supplement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids, ok := s.byProfile[profileID]
	if !ok {
		return []storage.Supplement{}, nil
	}

	var result []storage.Supplement
	for _, id := range ids {
		if supplement, ok := s.supplements[id]; ok {
			result = append(result, *supplement)
		}
	}

	return result, nil
}

func (s *SupplementsMemoryStorage) UpdateSupplement(ctx context.Context, supplement *storage.Supplement) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.supplements[supplement.ID]; !ok {
		return ErrNotFound
	}

	supplement.UpdatedAt = time.Now()
	clone := *supplement
	s.supplements[clone.ID] = &clone

	return nil
}

func (s *SupplementsMemoryStorage) DeleteSupplement(ctx context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	supplement, ok := s.supplements[id]
	if !ok {
		return ErrNotFound
	}

	// Remove from profile index
	profileID := supplement.ProfileID
	if ids, ok := s.byProfile[profileID]; ok {
		newIds := []uuid.UUID{}
		for _, sid := range ids {
			if sid != id {
				newIds = append(newIds, sid)
			}
		}
		s.byProfile[profileID] = newIds
	}

	delete(s.supplements, id)
	delete(s.components, id)

	return nil
}

func (s *SupplementsMemoryStorage) GetSupplementComponents(ctx context.Context, supplementID uuid.UUID) ([]storage.SupplementComponent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if components, ok := s.components[supplementID]; ok {
		result := make([]storage.SupplementComponent, len(components))
		copy(result, components)
		return result, nil
	}

	return []storage.SupplementComponent{}, nil
}

func (s *SupplementsMemoryStorage) SetSupplementComponents(ctx context.Context, supplementID uuid.UUID, components []storage.SupplementComponent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check supplement exists
	if _, ok := s.supplements[supplementID]; !ok {
		return ErrNotFound
	}

	// Set components (replace all)
	clones := make([]storage.SupplementComponent, len(components))
	for i, c := range components {
		if c.ID == uuid.Nil {
			c.ID = uuid.New()
		}
		if c.CreatedAt.IsZero() {
			c.CreatedAt = time.Now()
		}
		c.SupplementID = supplementID
		clones[i] = c
	}

	s.components[supplementID] = clones

	return nil
}

// IntakesMemoryStorage — in-memory storage for intakes
type IntakesMemoryStorage struct {
	mu                 sync.RWMutex
	waterIntakes       map[uuid.UUID]*storage.WaterIntake
	supplementIntakes  map[uuid.UUID]*storage.SupplementIntake
	waterByProfile     map[uuid.UUID][]uuid.UUID              // profile_id -> water_intake_ids
	supplementByProfile map[uuid.UUID][]uuid.UUID             // profile_id -> supplement_intake_ids
	supplementUnique   map[string]uuid.UUID                   // unique key -> supplement_intake_id
}

func NewIntakesMemoryStorage() *IntakesMemoryStorage {
	return &IntakesMemoryStorage{
		waterIntakes:        make(map[uuid.UUID]*storage.WaterIntake),
		supplementIntakes:   make(map[uuid.UUID]*storage.SupplementIntake),
		waterByProfile:      make(map[uuid.UUID][]uuid.UUID),
		supplementByProfile: make(map[uuid.UUID][]uuid.UUID),
		supplementUnique:    make(map[string]uuid.UUID),
	}
}

func (s *IntakesMemoryStorage) AddWater(ctx context.Context, profileID uuid.UUID, takenAt time.Time, amountMl int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	intake := &storage.WaterIntake{
		ID:        uuid.New(),
		ProfileID: profileID,
		TakenAt:   takenAt,
		AmountMl:  amountMl,
		CreatedAt: time.Now(),
	}

	s.waterIntakes[intake.ID] = intake
	s.waterByProfile[profileID] = append(s.waterByProfile[profileID], intake.ID)

	return nil
}

func (s *IntakesMemoryStorage) GetWaterDaily(ctx context.Context, profileID uuid.UUID, date string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids, ok := s.waterByProfile[profileID]
	if !ok {
		return 0, nil
	}

	total := 0
	for _, id := range ids {
		if intake, ok := s.waterIntakes[id]; ok {
			intakeDate := intake.TakenAt.Format("2006-01-02")
			if intakeDate == date {
				total += intake.AmountMl
			}
		}
	}

	return total, nil
}

func (s *IntakesMemoryStorage) ListWaterIntakes(ctx context.Context, profileID uuid.UUID, date string, limit int) ([]storage.WaterIntake, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids, ok := s.waterByProfile[profileID]
	if !ok {
		return []storage.WaterIntake{}, nil
	}

	var result []storage.WaterIntake
	for _, id := range ids {
		if intake, ok := s.waterIntakes[id]; ok {
			intakeDate := intake.TakenAt.Format("2006-01-02")
			if intakeDate == date {
				result = append(result, *intake)
			}
		}
	}

	// Sort by taken_at desc
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].TakenAt.Before(result[j].TakenAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

func (s *IntakesMemoryStorage) UpsertSupplementIntake(ctx context.Context, intake *storage.SupplementIntake) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate unique key
	date := intake.TakenAt.Format("2006-01-02")
	uniqueKey := intake.ProfileID.String() + ":" + intake.SupplementID.String() + ":" + date

	// Check if exists
	if existingID, exists := s.supplementUnique[uniqueKey]; exists {
		if existing, ok := s.supplementIntakes[existingID]; ok {
			existing.Status = intake.Status
			existing.TakenAt = intake.TakenAt
			return nil
		}
	}

	// Create new
	if intake.ID == uuid.Nil {
		intake.ID = uuid.New()
	}
	if intake.CreatedAt.IsZero() {
		intake.CreatedAt = time.Now()
	}

	clone := *intake
	s.supplementIntakes[clone.ID] = &clone
	s.supplementByProfile[clone.ProfileID] = append(s.supplementByProfile[clone.ProfileID], clone.ID)
	s.supplementUnique[uniqueKey] = clone.ID

	return nil
}

func (s *IntakesMemoryStorage) ListSupplementIntakes(ctx context.Context, profileID uuid.UUID, from, to string) ([]storage.SupplementIntake, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids, ok := s.supplementByProfile[profileID]
	if !ok {
		return []storage.SupplementIntake{}, nil
	}

	var result []storage.SupplementIntake
	for _, id := range ids {
		if intake, ok := s.supplementIntakes[id]; ok {
			intakeDate := intake.TakenAt.Format("2006-01-02")
			if intakeDate >= from && intakeDate <= to {
				result = append(result, *intake)
			}
		}
	}

	return result, nil
}

func (s *IntakesMemoryStorage) GetSupplementDaily(ctx context.Context, profileID uuid.UUID, date string) (map[uuid.UUID]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids, ok := s.supplementByProfile[profileID]
	if !ok {
		return map[uuid.UUID]string{}, nil
	}

	result := make(map[uuid.UUID]string)
	for _, id := range ids {
		if intake, ok := s.supplementIntakes[id]; ok {
			intakeDate := intake.TakenAt.Format("2006-01-02")
			if intakeDate == date {
				result[intake.SupplementID] = intake.Status
			}
		}
	}

	return result, nil
}
