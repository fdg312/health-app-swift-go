package foodprefs

import (
	"context"
	"fmt"

	"github.com/fdg312/health-hub/internal/storage"
)

const maxFoodPrefs = 200

// Service handles food preferences business logic.
type Service struct {
	storage storage.FoodPrefsStorage
}

// NewService creates a new food preferences service.
func NewService(storage storage.FoodPrefsStorage) *Service {
	return &Service{storage: storage}
}

// List returns food preferences with optional search query.
func (s *Service) List(ctx context.Context, ownerUserID string, profileID string, query string, limit, offset int) ([]storage.FoodPref, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	return s.storage.List(ctx, ownerUserID, profileID, query, limit, offset)
}

// Upsert creates or updates a food preference.
func (s *Service) Upsert(ctx context.Context, ownerUserID string, profileID string, req UpsertFoodPrefRequest) (storage.FoodPref, error) {
	if err := req.Validate(); err != nil {
		return storage.FoodPref{}, fmt.Errorf("validation failed: %w", err)
	}

	// Check max count when creating new
	if req.ID == "" {
		existing, total, err := s.storage.List(ctx, ownerUserID, profileID, "", 1, 0)
		if err != nil {
			return storage.FoodPref{}, fmt.Errorf("failed to check existing count: %w", err)
		}
		_ = existing

		if total >= maxFoodPrefs {
			return storage.FoodPref{}, fmt.Errorf("maximum food preferences limit reached (%d)", maxFoodPrefs)
		}
	}

	upsert := storage.FoodPrefUpsert{
		ID:              req.ID,
		Name:            req.Name,
		Tags:            req.Tags,
		KcalPer100g:     req.KcalPer100g,
		ProteinGPer100g: req.ProteinGPer100g,
		FatGPer100g:     req.FatGPer100g,
		CarbsGPer100g:   req.CarbsGPer100g,
	}

	return s.storage.Upsert(ctx, ownerUserID, profileID, upsert)
}

// Delete removes a food preference.
func (s *Service) Delete(ctx context.Context, ownerUserID string, id string) error {
	return s.storage.Delete(ctx, ownerUserID, id)
}
