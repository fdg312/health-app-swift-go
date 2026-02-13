package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresChatStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresChatStorage(pool *pgxpool.Pool) *PostgresChatStorage {
	return &PostgresChatStorage{pool: pool}
}

func (s *PostgresChatStorage) InsertMessage(ctx context.Context, ownerUserID string, profileID uuid.UUID, role, content string) (storage.ChatMessage, error) {
	msg := storage.ChatMessage{
		ID:          uuid.New(),
		OwnerUserID: strings.TrimSpace(ownerUserID),
		ProfileID:   profileID,
		Role:        strings.TrimSpace(role),
		Content:     content,
		CreatedAt:   time.Now().UTC(),
	}

	const query = `
		INSERT INTO chat_messages (id, owner_user_id, profile_id, role, content, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.pool.Exec(ctx, query,
		msg.ID,
		msg.OwnerUserID,
		msg.ProfileID,
		msg.Role,
		msg.Content,
		msg.CreatedAt,
	)
	if err != nil {
		return storage.ChatMessage{}, err
	}
	return msg, nil
}

func (s *PostgresChatStorage) ListMessages(ctx context.Context, ownerUserID string, profileID uuid.UUID, limit int, before *time.Time) ([]storage.ChatMessage, *time.Time, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if limit <= 0 {
		limit = 50
	}
	queryLimit := limit + 1

	const query = `
		SELECT id, owner_user_id, profile_id, role, content, created_at
		FROM (
			SELECT id, owner_user_id, profile_id, role, content, created_at
			FROM chat_messages
			WHERE owner_user_id = $1
			  AND profile_id = $2
			  AND ($3::timestamptz IS NULL OR created_at < $3)
			ORDER BY created_at DESC, id DESC
			LIMIT $4
		) latest
		ORDER BY created_at ASC, id ASC
	`

	rows, err := s.pool.Query(ctx, query, ownerUserID, profileID, before, queryLimit)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	result := make([]storage.ChatMessage, 0, queryLimit)
	for rows.Next() {
		var msg storage.ChatMessage
		if err := rows.Scan(
			&msg.ID,
			&msg.OwnerUserID,
			&msg.ProfileID,
			&msg.Role,
			&msg.Content,
			&msg.CreatedAt,
		); err != nil {
			return nil, nil, err
		}
		result = append(result, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	if len(result) <= limit {
		return result, nil, nil
	}

	result = result[1:]
	cursor := result[0].CreatedAt.UTC()
	return result, &cursor, nil
}
