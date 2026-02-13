package nutrition

import (
	"context"
	"fmt"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

// Service handles nutrition targets business logic.
type Service struct {
	storage        storage.Storage
	targetsStorage storage.NutritionTargetsStorage
}

// NewService creates a new nutrition service.
func NewService(storage storage.Storage, targetsStorage storage.NutritionTargetsStorage) *Service {
	return &Service{
		storage:        storage,
		targetsStorage: targetsStorage,
	}
}

// GetOrDefault returns nutrition targets for a profile or defaults if not set.
// Performs ownership check - returns error if profile doesn't belong to user.
func (s *Service) GetOrDefault(ctx context.Context, ownerUserID string, profileID uuid.UUID) (TargetsDTO, bool, error) {
	// Check profile ownership
	profile, err := s.storage.GetProfile(ctx, profileID)
	if err != nil {
		return TargetsDTO{}, false, fmt.Errorf("failed to get profile: %w", err)
	}
	if profile == nil || profile.OwnerUserID != ownerUserID {
		return TargetsDTO{}, false, fmt.Errorf("profile_not_found")
	}

	// Try to get existing targets
	target, err := s.targetsStorage.Get(ctx, ownerUserID, profileID)
	if err != nil {
		return TargetsDTO{}, false, fmt.Errorf("failed to get nutrition targets: %w", err)
	}

	if target == nil {
		// Return defaults
		defaults := GetDefaultTargets(profileID)
		return defaults, true, nil
	}

	// Return existing targets
	dto := TargetsDTO{
		ProfileID:    target.ProfileID,
		CaloriesKcal: target.CaloriesKcal,
		ProteinG:     target.ProteinG,
		FatG:         target.FatG,
		CarbsG:       target.CarbsG,
		CalciumMg:    target.CalciumMg,
		CreatedAt:    target.CreatedAt,
		UpdatedAt:    target.UpdatedAt,
	}

	return dto, false, nil
}

// Upsert creates or updates nutrition targets for a profile.
// Performs ownership check - returns error if profile doesn't belong to user.
func (s *Service) Upsert(ctx context.Context, ownerUserID string, req UpsertTargetsRequest) (TargetsDTO, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return TargetsDTO{}, fmt.Errorf("invalid_request: %w", err)
	}

	// Check profile ownership
	profile, err := s.storage.GetProfile(ctx, req.ProfileID)
	if err != nil {
		return TargetsDTO{}, fmt.Errorf("failed to get profile: %w", err)
	}
	if profile == nil || profile.OwnerUserID != ownerUserID {
		return TargetsDTO{}, fmt.Errorf("profile_not_found")
	}

	// Upsert targets
	upsert := storage.NutritionTargetUpsert{
		CaloriesKcal: req.CaloriesKcal,
		ProteinG:     req.ProteinG,
		FatG:         req.FatG,
		CarbsG:       req.CarbsG,
		CalciumMg:    req.CalciumMg,
	}

	target, err := s.targetsStorage.Upsert(ctx, ownerUserID, req.ProfileID, upsert)
	if err != nil {
		return TargetsDTO{}, fmt.Errorf("failed to upsert nutrition targets: %w", err)
	}

	dto := TargetsDTO{
		ProfileID:    target.ProfileID,
		CaloriesKcal: target.CaloriesKcal,
		ProteinG:     target.ProteinG,
		FatG:         target.FatG,
		CarbsG:       target.CarbsG,
		CalciumMg:    target.CalciumMg,
		CreatedAt:    target.CreatedAt,
		UpdatedAt:    target.UpdatedAt,
	}

	return dto, nil
}

// UpsertSimple creates or updates nutrition targets with individual parameters.
// Used by proposals service. Performs ownership check.
func (s *Service) UpsertSimple(ctx context.Context, ownerUserID string, profileID uuid.UUID, caloriesKcal, proteinG, fatG, carbsG, calciumMg int) error {
	req := UpsertTargetsRequest{
		ProfileID:    profileID,
		CaloriesKcal: caloriesKcal,
		ProteinG:     proteinG,
		FatG:         fatG,
		CarbsG:       carbsG,
		CalciumMg:    calciumMg,
	}
	_, err := s.Upsert(ctx, ownerUserID, req)
	return err
}
