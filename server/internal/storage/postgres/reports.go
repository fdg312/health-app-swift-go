package postgres

import (
	"context"
	"fmt"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresReportsStorage — Postgres storage для отчётов
type PostgresReportsStorage struct {
	pool *pgxpool.Pool
}

// NewPostgresReportsStorage создаёт новое Postgres хранилище
func NewPostgresReportsStorage(pool *pgxpool.Pool) *PostgresReportsStorage {
	return &PostgresReportsStorage{pool: pool}
}

// CreateReport создаёт новый отчёт
func (s *PostgresReportsStorage) CreateReport(ctx context.Context, report *storage.ReportMeta) error {
	query := `
		INSERT INTO reports (id, profile_id, format, from_date, to_date, object_key, size_bytes, status, error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	if report.ID == uuid.Nil {
		report.ID = uuid.New()
	}

	err := s.pool.QueryRow(ctx, query,
		report.ID,
		report.ProfileID,
		report.Format,
		report.FromDate,
		report.ToDate,
		report.ObjectKey,
		report.SizeBytes,
		report.Status,
		report.Error,
	).Scan(&report.CreatedAt, &report.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create report: %w", err)
	}

	return nil
}

// GetReport возвращает отчёт по ID
func (s *PostgresReportsStorage) GetReport(ctx context.Context, id uuid.UUID) (*storage.ReportMeta, error) {
	query := `
		SELECT id, profile_id, format, from_date, to_date, object_key, size_bytes, status, error, created_at, updated_at
		FROM reports
		WHERE id = $1
	`

	var report storage.ReportMeta
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&report.ID,
		&report.ProfileID,
		&report.Format,
		&report.FromDate,
		&report.ToDate,
		&report.ObjectKey,
		&report.SizeBytes,
		&report.Status,
		&report.Error,
		&report.CreatedAt,
		&report.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("report not found: %w", err)
	}

	return &report, nil
}

// ListReports возвращает список отчётов с пагинацией
func (s *PostgresReportsStorage) ListReports(ctx context.Context, profileID uuid.UUID, limit, offset int) ([]storage.ReportMeta, error) {
	query := `
		SELECT id, profile_id, format, from_date, to_date, object_key, size_bytes, status, error, created_at, updated_at
		FROM reports
		WHERE profile_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.pool.Query(ctx, query, profileID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list reports: %w", err)
	}
	defer rows.Close()

	var reports []storage.ReportMeta
	for rows.Next() {
		var r storage.ReportMeta
		err := rows.Scan(
			&r.ID,
			&r.ProfileID,
			&r.Format,
			&r.FromDate,
			&r.ToDate,
			&r.ObjectKey,
			&r.SizeBytes,
			&r.Status,
			&r.Error,
			&r.CreatedAt,
			&r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan report: %w", err)
		}
		reports = append(reports, r)
	}

	return reports, nil
}

// DeleteReport удаляет отчёт
func (s *PostgresReportsStorage) DeleteReport(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM reports WHERE id = $1`
	result, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete report: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("report not found")
	}

	return nil
}
