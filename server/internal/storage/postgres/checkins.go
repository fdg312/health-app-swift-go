package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/fdg312/health-hub/internal/checkins"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresCheckinsStorage implements checkins.Storage
type PostgresCheckinsStorage struct {
	pool *pgxpool.Pool
}

// NewPostgresCheckinsStorage creates a new Postgres checkins storage
func NewPostgresCheckinsStorage(pool *pgxpool.Pool) *PostgresCheckinsStorage {
	return &PostgresCheckinsStorage{pool: pool}
}

// ListCheckins returns all check-ins for a profile within a date range
func (s *PostgresCheckinsStorage) ListCheckins(profileID uuid.UUID, from, to string) ([]checkins.Checkin, error) {
	query := `
		SELECT id, profile_id, date, type, score, tags, note, created_at, updated_at
		FROM checkins
		WHERE profile_id = $1 AND date >= $2 AND date <= $3
		ORDER BY date DESC, type
	`

	rows, err := s.pool.Query(context.Background(), query, profileID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []checkins.Checkin
	for rows.Next() {
		var c checkins.Checkin
		var tagsJSON []byte

		err := rows.Scan(
			&c.ID,
			&c.ProfileID,
			&c.Date,
			&c.Type,
			&c.Score,
			&tagsJSON,
			&c.Note,
			&c.CreatedAt,
			&c.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal tags
		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &c.Tags); err != nil {
				return nil, err
			}
		}

		result = append(result, c)
	}

	return result, rows.Err()
}

// GetCheckin retrieves a check-in by ID
func (s *PostgresCheckinsStorage) GetCheckin(id uuid.UUID) (*checkins.Checkin, error) {
	query := `
		SELECT id, profile_id, date, type, score, tags, note, created_at, updated_at
		FROM checkins
		WHERE id = $1
	`

	var c checkins.Checkin
	var tagsJSON []byte

	err := s.pool.QueryRow(context.Background(), query, id).Scan(
		&c.ID,
		&c.ProfileID,
		&c.Date,
		&c.Type,
		&c.Score,
		&tagsJSON,
		&c.Note,
		&c.CreatedAt,
		&c.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("checkin not found")
		}
		return nil, err
	}

	// Unmarshal tags
	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &c.Tags); err != nil {
			return nil, err
		}
	}

	return &c, nil
}

// UpsertCheckin creates or updates a check-in (by profile_id, date, type)
func (s *PostgresCheckinsStorage) UpsertCheckin(checkin *checkins.Checkin) error {
	tagsJSON, err := json.Marshal(checkin.Tags)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO checkins (id, profile_id, date, type, score, tags, note, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (profile_id, date, type)
		DO UPDATE SET
			score = EXCLUDED.score,
			tags = EXCLUDED.tags,
			note = EXCLUDED.note,
			updated_at = EXCLUDED.updated_at
		RETURNING id, created_at, updated_at
	`

	err = s.pool.QueryRow(
		context.Background(),
		query,
		checkin.ID,
		checkin.ProfileID,
		checkin.Date,
		checkin.Type,
		checkin.Score,
		tagsJSON,
		checkin.Note,
		checkin.CreatedAt,
		checkin.UpdatedAt,
	).Scan(&checkin.ID, &checkin.CreatedAt, &checkin.UpdatedAt)

	return err
}

// DeleteCheckin deletes a check-in by ID
func (s *PostgresCheckinsStorage) DeleteCheckin(id uuid.UUID) error {
	query := `DELETE FROM checkins WHERE id = $1`

	result, err := s.pool.Exec(context.Background(), query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("checkin not found")
	}

	return nil
}
