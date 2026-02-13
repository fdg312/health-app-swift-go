package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresSupplementsStorage — Postgres implementation for supplements
type PostgresSupplementsStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresSupplementsStorage(pool *pgxpool.Pool) *PostgresSupplementsStorage {
	return &PostgresSupplementsStorage{pool: pool}
}

func (s *PostgresSupplementsStorage) CreateSupplement(ctx context.Context, supplement *storage.Supplement) error {
	if supplement.ID == uuid.Nil {
		supplement.ID = uuid.New()
	}
	if supplement.CreatedAt.IsZero() {
		supplement.CreatedAt = time.Now()
		supplement.UpdatedAt = time.Now()
	}

	query := `
		INSERT INTO supplements (id, profile_id, name, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.pool.Exec(ctx, query,
		supplement.ID,
		supplement.ProfileID,
		supplement.Name,
		supplement.Notes,
		supplement.CreatedAt,
		supplement.UpdatedAt,
	)

	return err
}

func (s *PostgresSupplementsStorage) GetSupplement(ctx context.Context, id uuid.UUID) (*storage.Supplement, error) {
	query := `
		SELECT id, profile_id, name, notes, created_at, updated_at
		FROM supplements
		WHERE id = $1
	`

	var supplement storage.Supplement
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&supplement.ID,
		&supplement.ProfileID,
		&supplement.Name,
		&supplement.Notes,
		&supplement.CreatedAt,
		&supplement.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &supplement, nil
}

func (s *PostgresSupplementsStorage) ListSupplements(ctx context.Context, profileID uuid.UUID) ([]storage.Supplement, error) {
	query := `
		SELECT id, profile_id, name, notes, created_at, updated_at
		FROM supplements
		WHERE profile_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.pool.Query(ctx, query, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var supplements []storage.Supplement
	for rows.Next() {
		var supplement storage.Supplement
		if err := rows.Scan(
			&supplement.ID,
			&supplement.ProfileID,
			&supplement.Name,
			&supplement.Notes,
			&supplement.CreatedAt,
			&supplement.UpdatedAt,
		); err != nil {
			return nil, err
		}
		supplements = append(supplements, supplement)
	}

	return supplements, rows.Err()
}

func (s *PostgresSupplementsStorage) UpdateSupplement(ctx context.Context, supplement *storage.Supplement) error {
	supplement.UpdatedAt = time.Now()

	query := `
		UPDATE supplements
		SET name = $1, notes = $2, updated_at = $3
		WHERE id = $4
	`

	result, err := s.pool.Exec(ctx, query,
		supplement.Name,
		supplement.Notes,
		supplement.UpdatedAt,
		supplement.ID,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *PostgresSupplementsStorage) DeleteSupplement(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM supplements WHERE id = $1`

	result, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *PostgresSupplementsStorage) GetSupplementComponents(ctx context.Context, supplementID uuid.UUID) ([]storage.SupplementComponent, error) {
	query := `
		SELECT id, supplement_id, nutrient_key, hk_identifier, amount, unit, created_at
		FROM supplement_components
		WHERE supplement_id = $1
		ORDER BY created_at
	`

	rows, err := s.pool.Query(ctx, query, supplementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var components []storage.SupplementComponent
	for rows.Next() {
		var component storage.SupplementComponent
		if err := rows.Scan(
			&component.ID,
			&component.SupplementID,
			&component.NutrientKey,
			&component.HKIdentifier,
			&component.Amount,
			&component.Unit,
			&component.CreatedAt,
		); err != nil {
			return nil, err
		}
		components = append(components, component)
	}

	return components, rows.Err()
}

func (s *PostgresSupplementsStorage) SetSupplementComponents(ctx context.Context, supplementID uuid.UUID, components []storage.SupplementComponent) error {
	// Start transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete existing components
	_, err = tx.Exec(ctx, "DELETE FROM supplement_components WHERE supplement_id = $1", supplementID)
	if err != nil {
		return err
	}

	// Insert new components
	for _, component := range components {
		if component.ID == uuid.Nil {
			component.ID = uuid.New()
		}
		if component.CreatedAt.IsZero() {
			component.CreatedAt = time.Now()
		}

		query := `
			INSERT INTO supplement_components (id, supplement_id, nutrient_key, hk_identifier, amount, unit, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`

		_, err := tx.Exec(ctx, query,
			component.ID,
			supplementID,
			component.NutrientKey,
			component.HKIdentifier,
			component.Amount,
			component.Unit,
			component.CreatedAt,
		)

		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// PostgresIntakesStorage — Postgres implementation for intakes
type PostgresIntakesStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresIntakesStorage(pool *pgxpool.Pool) *PostgresIntakesStorage {
	return &PostgresIntakesStorage{pool: pool}
}

func (s *PostgresIntakesStorage) AddWater(ctx context.Context, profileID uuid.UUID, takenAt time.Time, amountMl int) error {
	query := `
		INSERT INTO water_intakes (id, profile_id, taken_at, amount_ml, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	id := uuid.New()
	createdAt := time.Now()

	_, err := s.pool.Exec(ctx, query, id, profileID, takenAt, amountMl, createdAt)
	return err
}

func (s *PostgresIntakesStorage) GetWaterDaily(ctx context.Context, profileID uuid.UUID, date string) (int, error) {
	query := `
		SELECT COALESCE(SUM(amount_ml), 0)
		FROM water_intakes
		WHERE profile_id = $1
			AND DATE(taken_at AT TIME ZONE 'UTC') = $2
	`

	var total int
	err := s.pool.QueryRow(ctx, query, profileID, date).Scan(&total)
	if err != nil {
		return 0, err
	}

	return total, nil
}

func (s *PostgresIntakesStorage) ListWaterIntakes(ctx context.Context, profileID uuid.UUID, date string, limit int) ([]storage.WaterIntake, error) {
	query := `
		SELECT id, profile_id, taken_at, amount_ml, created_at
		FROM water_intakes
		WHERE profile_id = $1
			AND DATE(taken_at AT TIME ZONE 'UTC') = $2
		ORDER BY taken_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.pool.Query(ctx, query, profileID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var intakes []storage.WaterIntake
	for rows.Next() {
		var intake storage.WaterIntake
		if err := rows.Scan(
			&intake.ID,
			&intake.ProfileID,
			&intake.TakenAt,
			&intake.AmountMl,
			&intake.CreatedAt,
		); err != nil {
			return nil, err
		}
		intakes = append(intakes, intake)
	}

	return intakes, rows.Err()
}

func (s *PostgresIntakesStorage) UpsertSupplementIntake(ctx context.Context, intake *storage.SupplementIntake) error {
	if intake.ID == uuid.Nil {
		intake.ID = uuid.New()
	}
	if intake.CreatedAt.IsZero() {
		intake.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO supplement_intakes (id, profile_id, supplement_id, taken_at, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (profile_id, supplement_id, DATE(taken_at AT TIME ZONE 'UTC'))
		DO UPDATE SET
			status = EXCLUDED.status,
			taken_at = EXCLUDED.taken_at
	`

	_, err := s.pool.Exec(ctx, query,
		intake.ID,
		intake.ProfileID,
		intake.SupplementID,
		intake.TakenAt,
		intake.Status,
		intake.CreatedAt,
	)

	return err
}

func (s *PostgresIntakesStorage) ListSupplementIntakes(ctx context.Context, profileID uuid.UUID, from, to string) ([]storage.SupplementIntake, error) {
	query := `
		SELECT id, profile_id, supplement_id, taken_at, status, created_at
		FROM supplement_intakes
		WHERE profile_id = $1
			AND DATE(taken_at AT TIME ZONE 'UTC') >= $2
			AND DATE(taken_at AT TIME ZONE 'UTC') <= $3
		ORDER BY taken_at DESC
	`

	rows, err := s.pool.Query(ctx, query, profileID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var intakes []storage.SupplementIntake
	for rows.Next() {
		var intake storage.SupplementIntake
		if err := rows.Scan(
			&intake.ID,
			&intake.ProfileID,
			&intake.SupplementID,
			&intake.TakenAt,
			&intake.Status,
			&intake.CreatedAt,
		); err != nil {
			return nil, err
		}
		intakes = append(intakes, intake)
	}

	return intakes, rows.Err()
}

func (s *PostgresIntakesStorage) GetSupplementDaily(ctx context.Context, profileID uuid.UUID, date string) (map[uuid.UUID]string, error) {
	query := `
		SELECT supplement_id, status
		FROM supplement_intakes
		WHERE profile_id = $1
			AND DATE(taken_at AT TIME ZONE 'UTC') = $2
	`

	rows, err := s.pool.Query(ctx, query, profileID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID]string)
	for rows.Next() {
		var supplementID uuid.UUID
		var status string
		if err := rows.Scan(&supplementID, &status); err != nil {
			return nil, err
		}
		result[supplementID] = status
	}

	return result, rows.Err()
}
