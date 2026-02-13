package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSourceNotFound = errors.New("source not found")
)

// PostgresSourcesStorage — Postgres реализация SourcesStorage
type PostgresSourcesStorage struct {
	pool *pgxpool.Pool
}

// NewPostgresSourcesStorage создаёт новый PostgresSourcesStorage
func NewPostgresSourcesStorage(pool *pgxpool.Pool) *PostgresSourcesStorage {
	return &PostgresSourcesStorage{pool: pool}
}

func (s *PostgresSourcesStorage) CreateSource(ctx context.Context, source *storage.Source) error {
	query := `
		INSERT INTO sources (
			id, profile_id, kind, title, text, url, checkin_id,
			object_key, content_type, size_bytes, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`

	if source.ID == uuid.Nil {
		source.ID = uuid.New()
	}

	now := time.Now()
	source.CreatedAt = now
	source.UpdatedAt = now

	_, err := s.pool.Exec(ctx, query,
		source.ID,
		source.ProfileID,
		source.Kind,
		source.Title,
		source.Text,
		source.URL,
		source.CheckinID,
		source.ObjectKey,
		source.ContentType,
		source.SizeBytes,
		source.CreatedAt,
		source.UpdatedAt,
	)

	return err
}

func (s *PostgresSourcesStorage) GetSource(ctx context.Context, id uuid.UUID) (*storage.Source, error) {
	query := `
		SELECT id, profile_id, kind, title, text, url, checkin_id,
		       object_key, content_type, size_bytes, created_at, updated_at
		FROM sources
		WHERE id = $1
	`

	var src storage.Source
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&src.ID,
		&src.ProfileID,
		&src.Kind,
		&src.Title,
		&src.Text,
		&src.URL,
		&src.CheckinID,
		&src.ObjectKey,
		&src.ContentType,
		&src.SizeBytes,
		&src.CreatedAt,
		&src.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSourceNotFound
		}
		return nil, err
	}

	return &src, nil
}

func (s *PostgresSourcesStorage) ListSources(ctx context.Context, profileID uuid.UUID, query string, checkinID *uuid.UUID, limit, offset int) ([]storage.Source, error) {
	// Build dynamic query with optional filters
	baseQuery := `
		SELECT id, profile_id, kind, title, text, url, checkin_id,
		       object_key, content_type, size_bytes, created_at, updated_at
		FROM sources
		WHERE profile_id = $1
	`

	args := []interface{}{profileID}
	argCount := 1

	if checkinID != nil {
		argCount++
		baseQuery += fmt.Sprintf(` AND checkin_id = $%d`, argCount)
		args = append(args, *checkinID)
	}

	if query != "" {
		argCount++
		baseQuery += fmt.Sprintf(` AND (
			title ILIKE $%d OR
			text ILIKE $%d OR
			url ILIKE $%d
		)`, argCount, argCount, argCount)
		searchPattern := "%" + query + "%"
		args = append(args, searchPattern)
	}

	argCount++
	argCount++
	baseQuery += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argCount-1, argCount)
	args = append(args, limit, offset)

	rows, err := s.pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []storage.Source
	for rows.Next() {
		var src storage.Source
		err := rows.Scan(
			&src.ID,
			&src.ProfileID,
			&src.Kind,
			&src.Title,
			&src.Text,
			&src.URL,
			&src.CheckinID,
			&src.ObjectKey,
			&src.ContentType,
			&src.SizeBytes,
			&src.CreatedAt,
			&src.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		sources = append(sources, src)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sources, nil
}

func (s *PostgresSourcesStorage) DeleteSource(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM sources WHERE id = $1`

	result, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrSourceNotFound
	}

	return nil
}

// GetSourceBlob - for Postgres, we don't store blobs in DB in S3 mode.
// This is only used in memory mode, so return not found for Postgres.
func (s *PostgresSourcesStorage) GetSourceBlob(ctx context.Context, sourceID uuid.UUID) ([]byte, string, error) {
	// Not implemented for Postgres - blobs are stored in S3 or memory
	return nil, "", errors.New("blob storage not available in postgres mode")
}

// PutSourceBlob - same as above, not used in Postgres mode
func (s *PostgresSourcesStorage) PutSourceBlob(ctx context.Context, sourceID uuid.UUID, data []byte, contentType string) error {
	// Not implemented for Postgres - blobs are stored in S3 or memory
	return errors.New("blob storage not available in postgres mode")
}
