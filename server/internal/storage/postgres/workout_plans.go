package postgres

import (
	"context"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresWorkoutPlansStorage implements workout plans storage for Postgres.
type PostgresWorkoutPlansStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresWorkoutPlansStorage(pool *pgxpool.Pool) *PostgresWorkoutPlansStorage {
	return &PostgresWorkoutPlansStorage{pool: pool}
}

// GetActivePlan returns the active workout plan for a profile.
func (s *PostgresWorkoutPlansStorage) GetActivePlan(ownerUserID string, profileID uuid.UUID) (storage.WorkoutPlan, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, owner_user_id, profile_id, title, goal, is_active, created_at, updated_at
		FROM workout_plans
		WHERE owner_user_id = $1 AND profile_id = $2 AND is_active = true
		LIMIT 1
	`

	var plan storage.WorkoutPlan
	err := s.pool.QueryRow(ctx, query, ownerUserID, profileID).Scan(
		&plan.ID,
		&plan.OwnerUserID,
		&plan.ProfileID,
		&plan.Title,
		&plan.Goal,
		&plan.IsActive,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return storage.WorkoutPlan{}, false, nil
		}
		return storage.WorkoutPlan{}, false, err
	}

	// Verify ownership (paranoid check)
	if plan.OwnerUserID != ownerUserID || plan.ProfileID != profileID {
		return storage.WorkoutPlan{}, false, nil
	}

	return plan, true, nil
}

// UpsertActivePlan creates or updates the active workout plan.
func (s *PostgresWorkoutPlansStorage) UpsertActivePlan(ownerUserID string, profileID uuid.UUID, title string, goal string) (storage.WorkoutPlan, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return storage.WorkoutPlan{}, err
	}
	defer tx.Rollback(ctx)

	// Deactivate existing active plans
	_, err = tx.Exec(ctx, `
		UPDATE workout_plans
		SET is_active = false, updated_at = now()
		WHERE owner_user_id = $1 AND profile_id = $2 AND is_active = true
	`, ownerUserID, profileID)
	if err != nil {
		return storage.WorkoutPlan{}, err
	}

	// Create new active plan
	query := `
		INSERT INTO workout_plans (owner_user_id, profile_id, title, goal, is_active)
		VALUES ($1, $2, $3, $4, true)
		RETURNING id, owner_user_id, profile_id, title, goal, is_active, created_at, updated_at
	`

	var plan storage.WorkoutPlan
	err = tx.QueryRow(ctx, query, ownerUserID, profileID, title, goal).Scan(
		&plan.ID,
		&plan.OwnerUserID,
		&plan.ProfileID,
		&plan.Title,
		&plan.Goal,
		&plan.IsActive,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)
	if err != nil {
		return storage.WorkoutPlan{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return storage.WorkoutPlan{}, err
	}

	return plan, nil
}
