package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

type ChatMemoryStorage struct {
	mu       sync.RWMutex
	messages []storage.ChatMessage
}

func NewChatMemoryStorage() *ChatMemoryStorage {
	return &ChatMemoryStorage{
		messages: make([]storage.ChatMessage, 0),
	}
}

func (s *ChatMemoryStorage) InsertMessage(ctx context.Context, ownerUserID string, profileID uuid.UUID, role, content string) (storage.ChatMessage, error) {
	_ = ctx

	s.mu.Lock()
	defer s.mu.Unlock()

	msg := storage.ChatMessage{
		ID:          uuid.New(),
		OwnerUserID: strings.TrimSpace(ownerUserID),
		ProfileID:   profileID,
		Role:        strings.TrimSpace(role),
		Content:     content,
		CreatedAt:   time.Now().UTC(),
	}

	s.messages = append(s.messages, msg)
	return msg, nil
}

func (s *ChatMemoryStorage) ListMessages(ctx context.Context, ownerUserID string, profileID uuid.UUID, limit int, before *time.Time) ([]storage.ChatMessage, *time.Time, error) {
	_ = ctx

	ownerUserID = strings.TrimSpace(ownerUserID)
	if limit <= 0 {
		limit = 50
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]storage.ChatMessage, 0, len(s.messages))
	for _, msg := range s.messages {
		if msg.OwnerUserID != ownerUserID || msg.ProfileID != profileID {
			continue
		}
		if before != nil && !msg.CreatedAt.Before(*before) {
			continue
		}
		filtered = append(filtered, msg)
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].CreatedAt.Equal(filtered[j].CreatedAt) {
			return filtered[i].ID.String() < filtered[j].ID.String()
		}
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})

	if len(filtered) <= limit {
		return filtered, nil, nil
	}

	messages := filtered[len(filtered)-limit:]
	cursor := messages[0].CreatedAt.UTC()
	return messages, &cursor, nil
}
