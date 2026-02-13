package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresEmailOTPStorage stores OTP codes in PostgreSQL.
type PostgresEmailOTPStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresEmailOTPStorage(pool *pgxpool.Pool) *PostgresEmailOTPStorage {
	return &PostgresEmailOTPStorage{pool: pool}
}

func (s *PostgresEmailOTPStorage) CreateOrReplace(ctx context.Context, email, codeHash string, expiresAt, now time.Time, maxAttempts int) (uuid.UUID, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, `DELETE FROM email_otps WHERE email = $1 AND expires_at > $2`, email, now)
	if err != nil {
		return uuid.Nil, err
	}

	otpID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO email_otps (id, email, code_hash, created_at, expires_at, attempts, max_attempts, last_sent_at, send_count)
		VALUES ($1, $2, $3, $4, $5, 0, $6, $4, 1)
	`, otpID, email, codeHash, now, expiresAt, maxAttempts)
	if err != nil {
		return uuid.Nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}

	return otpID, nil
}

func (s *PostgresEmailOTPStorage) GetLatestActive(ctx context.Context, email string, now time.Time) (*storage.EmailOTP, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	query := `
		SELECT id, email, code_hash, created_at, expires_at, attempts, max_attempts, last_sent_at, send_count
		FROM email_otps
		WHERE email = $1 AND expires_at > $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	var row storage.EmailOTP
	err := s.pool.QueryRow(ctx, query, email, now).Scan(
		&row.ID,
		&row.Email,
		&row.CodeHash,
		&row.CreatedAt,
		&row.ExpiresAt,
		&row.Attempts,
		&row.MaxAttempts,
		&row.LastSentAt,
		&row.SendCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (s *PostgresEmailOTPStorage) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE email_otps SET attempts = attempts + 1 WHERE id = $1`, id)
	return err
}

func (s *PostgresEmailOTPStorage) MarkUsedOrDelete(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM email_otps WHERE id = $1`, id)
	return err
}

func (s *PostgresEmailOTPStorage) UpdateResendMeta(ctx context.Context, id uuid.UUID, lastSentAt time.Time, sendCount int) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE email_otps
		SET last_sent_at = $2, send_count = $3
		WHERE id = $1
	`, id, lastSentAt, sendCount)
	return err
}
