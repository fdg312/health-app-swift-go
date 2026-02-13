package postgres

import (
	"context"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresWorkoutCompletionsStorage implements workout completions storage for Postgres.
type PostgresWorkoutCompletionsStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresWorkoutCompletionsStorage(pool *pgxpool.Pool) *PostgresWorkoutCompletionsStorage {
	return &PostgresWorkoutCompletionsStorage{pool: pool}
}

// UpsertCompletion creates or updates a workout completion record.
func (s *PostgresWorkoutCompletionsStorage) UpsertCompletion(ownerUserID string, profileID uuid.UUID, date string, planItemID uuid.UUID, status string, note string) (storage.WorkoutCompletion, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO workout_completions (owner_user_id, profile_id, date, plan_item_id, status, note)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (owner_user_id, profile_id, date, plan_item_id)
		DO UPDATE SET
			status = EXCLUDED.status,
			note = EXCLUDED.note,
			updated_at = now()
		RETURNING id, owner_user_id, profile_id, date, plan_item_id, status, note, created_at, updated_at
	`

	var completion storage.WorkoutCompletion
	err := s.pool.QueryRow(ctx, query, ownerUserID, profileID, date, planItemID, status, note).Scan(
		&completion.ID,
		&completion.OwnerUserID,
		&completion.ProfileID,
		&completion.Date,
		&completion.PlanItemID,
		&completion.Status,
		&completion.Note,
		&completion.CreatedAt,
		&completion.UpdatedAt,
	)

	if err != nil {
		return storage.WorkoutCompletion{}, err
	}

	return completion, nil
}

// ListCompletions returns workout completion records in a date range.
func (s *PostgresWorkoutCompletionsStorage) ListCompletions(ownerUserID string, profileID uuid.UUID, from string, to string) ([]storage.WorkoutCompletion, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, owner_user_id, profile_id, date, plan_item_id, status, note, created_at, updated_at
		FROM workout_completions
		WHERE owner_user_id = $1 AND profile_id = $2 AND date >= $3 AND date <= $4
		ORDER BY date DESC, created_at DESC
	`

	rows, err := s.pool.Query(ctx, query, ownerUserID, profileID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var completions []storage.WorkoutCompletion
	for rows.Next() {
		var c storage.WorkoutCompletion
		err := rows.Scan(
			&c.ID,
			&c.OwnerUserID,
			&c.ProfileID,
			&c.Date,
			&c.PlanItemID,
			&c.Status,
			&c.Note,
			&c.CreatedAt,
			&c.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		completions = append(completions, c)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if completions == nil {
		completions = []storage.WorkoutCompletion{}
	}

	return completions, nil
}
