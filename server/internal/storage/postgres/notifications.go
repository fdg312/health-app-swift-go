package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresNotificationsStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresNotificationsStorage(pool *pgxpool.Pool) *PostgresNotificationsStorage {
	return &PostgresNotificationsStorage{pool: pool}
}

func (s *PostgresNotificationsStorage) CreateNotification(ctx context.Context, n *storage.Notification) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO notifications (id, profile_id, kind, title, body, source_date, severity, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (profile_id, kind, source_date)
		DO UPDATE SET
			title = EXCLUDED.title,
			body = EXCLUDED.body,
			severity = EXCLUDED.severity
	`

	_, err := s.pool.Exec(ctx, query,
		n.ID,
		n.ProfileID,
		n.Kind,
		n.Title,
		n.Body,
		n.SourceDate,
		n.Severity,
		n.CreatedAt,
	)

	return err
}

func (s *PostgresNotificationsStorage) ListNotifications(ctx context.Context, profileID uuid.UUID, onlyUnread bool, limit, offset int) ([]storage.Notification, error) {
	query := `
		SELECT id, profile_id, kind, title, body, source_date, severity, created_at, read_at
		FROM notifications
		WHERE profile_id = $1
	`

	args := []interface{}{profileID}

	if onlyUnread {
		query += " AND read_at IS NULL"
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	if offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", offset)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []storage.Notification

	for rows.Next() {
		var n storage.Notification
		if err := rows.Scan(
			&n.ID,
			&n.ProfileID,
			&n.Kind,
			&n.Title,
			&n.Body,
			&n.SourceDate,
			&n.Severity,
			&n.CreatedAt,
			&n.ReadAt,
		); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return notifications, nil
}

func (s *PostgresNotificationsStorage) UnreadCount(ctx context.Context, profileID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM notifications
		WHERE profile_id = $1 AND read_at IS NULL
	`

	var count int
	err := s.pool.QueryRow(ctx, query, profileID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *PostgresNotificationsStorage) MarkRead(ctx context.Context, profileID uuid.UUID, ids []uuid.UUID) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	query := `
		UPDATE notifications
		SET read_at = $1
		WHERE profile_id = $2
			AND id = ANY($3)
			AND read_at IS NULL
	`

	now := time.Now()
	result, err := s.pool.Exec(ctx, query, now, profileID, ids)
	if err != nil {
		return 0, err
	}

	return int(result.RowsAffected()), nil
}

func (s *PostgresNotificationsStorage) MarkAllRead(ctx context.Context, profileID uuid.UUID) (int, error) {
	query := `
		UPDATE notifications
		SET read_at = $1
		WHERE profile_id = $2 AND read_at IS NULL
	`

	now := time.Now()
	result, err := s.pool.Exec(ctx, query, now, profileID)
	if err != nil {
		return 0, err
	}

	return int(result.RowsAffected()), nil
}
