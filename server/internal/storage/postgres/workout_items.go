package postgres

import (
	"context"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresWorkoutPlanItemsStorage implements workout plan items storage for Postgres.
type PostgresWorkoutPlanItemsStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresWorkoutPlanItemsStorage(pool *pgxpool.Pool) *PostgresWorkoutPlanItemsStorage {
	return &PostgresWorkoutPlanItemsStorage{pool: pool}
}

// ListItems returns all workout plan items for a plan (with ownership check).
func (s *PostgresWorkoutPlanItemsStorage) ListItems(ownerUserID string, profileID uuid.UUID, planID uuid.UUID) ([]storage.WorkoutPlanItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, plan_id, owner_user_id, profile_id, kind, time_minutes, days_mask,
		       duration_min, intensity, note, details, created_at, updated_at
		FROM workout_plan_items
		WHERE owner_user_id = $1 AND profile_id = $2 AND plan_id = $3
		ORDER BY time_minutes ASC, kind ASC
	`

	rows, err := s.pool.Query(ctx, query, ownerUserID, profileID, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []storage.WorkoutPlanItem
	for rows.Next() {
		var item storage.WorkoutPlanItem
		err := rows.Scan(
			&item.ID,
			&item.PlanID,
			&item.OwnerUserID,
			&item.ProfileID,
			&item.Kind,
			&item.TimeMinutes,
			&item.DaysMask,
			&item.DurationMin,
			&item.Intensity,
			&item.Note,
			&item.Details,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if items == nil {
		items = []storage.WorkoutPlanItem{}
	}

	return items, nil
}

// ReplaceAllItems atomically replaces all items for a workout plan.
func (s *PostgresWorkoutPlanItemsStorage) ReplaceAllItems(ownerUserID string, profileID uuid.UUID, planID uuid.UUID, items []storage.WorkoutItemUpsert) ([]storage.WorkoutPlanItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Delete existing items for this plan
	_, err = tx.Exec(ctx, `
		DELETE FROM workout_plan_items
		WHERE owner_user_id = $1 AND profile_id = $2 AND plan_id = $3
	`, ownerUserID, profileID, planID)
	if err != nil {
		return nil, err
	}

	// Insert new items
	var result []storage.WorkoutPlanItem

	insertQuery := `
		INSERT INTO workout_plan_items
		(plan_id, owner_user_id, profile_id, kind, time_minutes, days_mask, duration_min, intensity, note, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, plan_id, owner_user_id, profile_id, kind, time_minutes, days_mask,
		          duration_min, intensity, note, details, created_at, updated_at
	`

	for _, upsert := range items {
		var item storage.WorkoutPlanItem
		err = tx.QueryRow(ctx, insertQuery,
			planID,
			ownerUserID,
			profileID,
			upsert.Kind,
			upsert.TimeMinutes,
			upsert.DaysMask,
			upsert.DurationMin,
			upsert.Intensity,
			upsert.Note,
			upsert.Details,
		).Scan(
			&item.ID,
			&item.PlanID,
			&item.OwnerUserID,
			&item.ProfileID,
			&item.Kind,
			&item.TimeMinutes,
			&item.DaysMask,
			&item.DurationMin,
			&item.Intensity,
			&item.Note,
			&item.Details,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	if result == nil {
		result = []storage.WorkoutPlanItem{}
	}

	return result, nil
}

// DeleteItem deletes a workout plan item (with ownership check).
func (s *PostgresWorkoutPlanItemsStorage) DeleteItem(ownerUserID string, itemID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		DELETE FROM workout_plan_items
		WHERE owner_user_id = $1 AND id = $2
	`

	_, err := s.pool.Exec(ctx, query, ownerUserID, itemID)
	return err
}
