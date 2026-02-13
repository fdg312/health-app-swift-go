package postgres

import (
	"context"
	"fmt"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type nutritionTargetsStorage struct {
	pool *pgxpool.Pool
}

func newNutritionTargetsStorage(pool *pgxpool.Pool) *nutritionTargetsStorage {
	return &nutritionTargetsStorage{pool: pool}
}

func (s *nutritionTargetsStorage) Get(ctx context.Context, ownerUserID string, profileID uuid.UUID) (*storage.NutritionTarget, error) {
	query := `
		SELECT id, owner_user_id, profile_id, calories_kcal, protein_g, fat_g, carbs_g, calcium_mg, created_at, updated_at
		FROM nutrition_targets
		WHERE owner_user_id = $1 AND profile_id = $2
	`

	var target storage.NutritionTarget
	err := s.pool.QueryRow(ctx, query, ownerUserID, profileID).Scan(
		&target.ID,
		&target.OwnerUserID,
		&target.ProfileID,
		&target.CaloriesKcal,
		&target.ProteinG,
		&target.FatG,
		&target.CarbsG,
		&target.CalciumMg,
		&target.CreatedAt,
		&target.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil // not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get nutrition targets: %w", err)
	}

	return &target, nil
}

func (s *nutritionTargetsStorage) Upsert(ctx context.Context, ownerUserID string, profileID uuid.UUID, upsert storage.NutritionTargetUpsert) (*storage.NutritionTarget, error) {
	query := `
		INSERT INTO nutrition_targets (owner_user_id, profile_id, calories_kcal, protein_g, fat_g, carbs_g, calcium_mg)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (owner_user_id, profile_id)
		DO UPDATE SET
			calories_kcal = EXCLUDED.calories_kcal,
			protein_g = EXCLUDED.protein_g,
			fat_g = EXCLUDED.fat_g,
			carbs_g = EXCLUDED.carbs_g,
			calcium_mg = EXCLUDED.calcium_mg,
			updated_at = now()
		RETURNING id, owner_user_id, profile_id, calories_kcal, protein_g, fat_g, carbs_g, calcium_mg, created_at, updated_at
	`

	var target storage.NutritionTarget
	err := s.pool.QueryRow(
		ctx,
		query,
		ownerUserID,
		profileID,
		upsert.CaloriesKcal,
		upsert.ProteinG,
		upsert.FatG,
		upsert.CarbsG,
		upsert.CalciumMg,
	).Scan(
		&target.ID,
		&target.OwnerUserID,
		&target.ProfileID,
		&target.CaloriesKcal,
		&target.ProteinG,
		&target.FatG,
		&target.CarbsG,
		&target.CalciumMg,
		&target.CreatedAt,
		&target.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert nutrition targets: %w", err)
	}

	return &target, nil
}
