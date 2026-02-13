package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

type nutritionTargetsStorage struct {
	mu      sync.RWMutex
	targets map[string]*storage.NutritionTarget // key: "ownerUserID:profileID"
}

func newNutritionTargetsStorage() *nutritionTargetsStorage {
	return &nutritionTargetsStorage{
		targets: make(map[string]*storage.NutritionTarget),
	}
}

func (s *nutritionTargetsStorage) Get(ctx context.Context, ownerUserID string, profileID uuid.UUID) (*storage.NutritionTarget, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", ownerUserID, profileID.String())
	target, ok := s.targets[key]
	if !ok {
		return nil, nil // not found, return nil without error
	}

	// Return a copy
	copied := *target
	return &copied, nil
}

func (s *nutritionTargetsStorage) Upsert(ctx context.Context, ownerUserID string, profileID uuid.UUID, upsert storage.NutritionTargetUpsert) (*storage.NutritionTarget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s", ownerUserID, profileID.String())
	now := time.Now().UTC()

	existing, ok := s.targets[key]
	if ok {
		// Update existing
		existing.CaloriesKcal = upsert.CaloriesKcal
		existing.ProteinG = upsert.ProteinG
		existing.FatG = upsert.FatG
		existing.CarbsG = upsert.CarbsG
		existing.CalciumMg = upsert.CalciumMg
		existing.UpdatedAt = now

		copied := *existing
		return &copied, nil
	}

	// Create new
	target := &storage.NutritionTarget{
		ID:           uuid.New(),
		OwnerUserID:  ownerUserID,
		ProfileID:    profileID,
		CaloriesKcal: upsert.CaloriesKcal,
		ProteinG:     upsert.ProteinG,
		FatG:         upsert.FatG,
		CarbsG:       upsert.CarbsG,
		CalciumMg:    upsert.CalciumMg,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	s.targets[key] = target

	copied := *target
	return &copied, nil
}
