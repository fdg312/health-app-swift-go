package postgres

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresSupplementSchedulesStorage — Postgres implementation for supplement schedules.
type PostgresSupplementSchedulesStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresSupplementSchedulesStorage(pool *pgxpool.Pool) *PostgresSupplementSchedulesStorage {
	return &PostgresSupplementSchedulesStorage{pool: pool}
}

func (s *PostgresSupplementSchedulesStorage) ListSchedules(ctx context.Context, ownerUserID string, profileID uuid.UUID) ([]storage.SupplementSchedule, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)

	const query = `
		SELECT id, owner_user_id, profile_id, supplement_id, time_minutes, days_mask, is_enabled, created_at, updated_at
		FROM supplement_schedules
		WHERE owner_user_id = $1
		  AND profile_id = $2
		ORDER BY time_minutes ASC, created_at ASC, id ASC
	`

	rows, err := s.pool.Query(ctx, query, ownerUserID, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]storage.SupplementSchedule, 0)
	for rows.Next() {
		var row storage.SupplementSchedule
		if err := rows.Scan(
			&row.ID,
			&row.OwnerUserID,
			&row.ProfileID,
			&row.SupplementID,
			&row.TimeMinutes,
			&row.DaysMask,
			&row.IsEnabled,
			&row.CreatedAt,
			&row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	return result, rows.Err()
}

func (s *PostgresSupplementSchedulesStorage) UpsertSchedule(ctx context.Context, ownerUserID string, profileID uuid.UUID, item storage.ScheduleUpsert) (storage.SupplementSchedule, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)

	const query = `
		INSERT INTO supplement_schedules (
			id, owner_user_id, profile_id, supplement_id, time_minutes, days_mask, is_enabled, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		ON CONFLICT (owner_user_id, profile_id, supplement_id, time_minutes)
		DO UPDATE SET
			days_mask = EXCLUDED.days_mask,
			is_enabled = EXCLUDED.is_enabled,
			updated_at = EXCLUDED.updated_at
		RETURNING id, owner_user_id, profile_id, supplement_id, time_minutes, days_mask, is_enabled, created_at, updated_at
	`

	now := time.Now().UTC()
	var row storage.SupplementSchedule
	err := s.pool.QueryRow(ctx, query,
		uuid.New(),
		ownerUserID,
		profileID,
		item.SupplementID,
		item.TimeMinutes,
		item.DaysMask,
		item.IsEnabled,
		now,
	).Scan(
		&row.ID,
		&row.OwnerUserID,
		&row.ProfileID,
		&row.SupplementID,
		&row.TimeMinutes,
		&row.DaysMask,
		&row.IsEnabled,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		return storage.SupplementSchedule{}, err
	}
	return row, nil
}

func (s *PostgresSupplementSchedulesStorage) DeleteSchedule(ctx context.Context, ownerUserID string, scheduleID uuid.UUID) error {
	ownerUserID = strings.TrimSpace(ownerUserID)

	const query = `
		DELETE FROM supplement_schedules
		WHERE owner_user_id = $1
		  AND id = $2
	`

	res, err := s.pool.Exec(ctx, query, ownerUserID, scheduleID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresSupplementSchedulesStorage) ReplaceAll(ctx context.Context, ownerUserID string, profileID uuid.UUID, items []storage.ScheduleUpsert) ([]storage.SupplementSchedule, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	const deleteQuery = `
		DELETE FROM supplement_schedules
		WHERE owner_user_id = $1
		  AND profile_id = $2
	`
	if _, err := tx.Exec(ctx, deleteQuery, ownerUserID, profileID); err != nil {
		return nil, err
	}

	const upsertQuery = `
		INSERT INTO supplement_schedules (
			id, owner_user_id, profile_id, supplement_id, time_minutes, days_mask, is_enabled, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		ON CONFLICT (owner_user_id, profile_id, supplement_id, time_minutes)
		DO UPDATE SET
			days_mask = EXCLUDED.days_mask,
			is_enabled = EXCLUDED.is_enabled,
			updated_at = EXCLUDED.updated_at
		RETURNING id, owner_user_id, profile_id, supplement_id, time_minutes, days_mask, is_enabled, created_at, updated_at
	`

	now := time.Now().UTC()
	dedupe := make(map[string]storage.ScheduleUpsert, len(items))
	for _, item := range items {
		key := item.SupplementID.String() + ":" + strconv.Itoa(item.TimeMinutes)
		dedupe[key] = item
	}

	result := make([]storage.SupplementSchedule, 0, len(dedupe))
	for _, item := range dedupe {
		var row storage.SupplementSchedule
		err := tx.QueryRow(ctx, upsertQuery,
			uuid.New(),
			ownerUserID,
			profileID,
			item.SupplementID,
			item.TimeMinutes,
			item.DaysMask,
			item.IsEnabled,
			now,
		).Scan(
			&row.ID,
			&row.OwnerUserID,
			&row.ProfileID,
			&row.SupplementID,
			&row.TimeMinutes,
			&row.DaysMask,
			&row.IsEnabled,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].TimeMinutes == result[j].TimeMinutes {
			return result[i].CreatedAt.Before(result[j].CreatedAt)
		}
		return result[i].TimeMinutes < result[j].TimeMinutes
	})

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return result, nil
}
