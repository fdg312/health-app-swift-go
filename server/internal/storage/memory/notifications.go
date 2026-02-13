package memory

import (
	"context"
	"sync"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

type NotificationsMemoryStorage struct {
	mu            sync.RWMutex
	notifications map[uuid.UUID]*storage.Notification           // id -> notification
	byProfile     map[uuid.UUID][]uuid.UUID                     // profile_id -> []notification_ids
	uniqueKeys    map[string]uuid.UUID                          // unique key -> notification_id
}

func NewNotificationsMemoryStorage() *NotificationsMemoryStorage {
	return &NotificationsMemoryStorage{
		notifications: make(map[uuid.UUID]*storage.Notification),
		byProfile:     make(map[uuid.UUID][]uuid.UUID),
		uniqueKeys:    make(map[string]uuid.UUID),
	}
}

func (s *NotificationsMemoryStorage) CreateNotification(ctx context.Context, n *storage.Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate unique key for upsert
	uniqueKey := makeUniqueKey(n.ProfileID, n.Kind, n.SourceDate)

	// Check if notification already exists
	if existingID, exists := s.uniqueKeys[uniqueKey]; exists {
		// Update existing
		if existing, ok := s.notifications[existingID]; ok {
			existing.Title = n.Title
			existing.Body = n.Body
			existing.Severity = n.Severity
			// Don't update CreatedAt, ReadAt
			return nil
		}
	}

	// Create new
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}

	// Clone for storage
	clone := *n
	s.notifications[clone.ID] = &clone

	// Add to profile index
	s.byProfile[clone.ProfileID] = append(s.byProfile[clone.ProfileID], clone.ID)

	// Add to unique keys
	s.uniqueKeys[uniqueKey] = clone.ID

	return nil
}

func (s *NotificationsMemoryStorage) ListNotifications(ctx context.Context, profileID uuid.UUID, onlyUnread bool, limit, offset int) ([]storage.Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids, ok := s.byProfile[profileID]
	if !ok {
		return []storage.Notification{}, nil
	}

	var result []storage.Notification
	for _, id := range ids {
		if n, ok := s.notifications[id]; ok {
			if onlyUnread && n.ReadAt != nil {
				continue
			}
			result = append(result, *n)
		}
	}

	// Sort by created_at desc
	sortByCreatedAtDesc(result)

	// Apply pagination
	if offset >= len(result) {
		return []storage.Notification{}, nil
	}
	result = result[offset:]
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

func (s *NotificationsMemoryStorage) UnreadCount(ctx context.Context, profileID uuid.UUID) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids, ok := s.byProfile[profileID]
	if !ok {
		return 0, nil
	}

	count := 0
	for _, id := range ids {
		if n, ok := s.notifications[id]; ok {
			if n.ReadAt == nil {
				count++
			}
		}
	}

	return count, nil
}

func (s *NotificationsMemoryStorage) MarkRead(ctx context.Context, profileID uuid.UUID, ids []uuid.UUID) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	marked := 0
	now := time.Now()

	for _, id := range ids {
		if n, ok := s.notifications[id]; ok {
			// Check ownership
			if n.ProfileID != profileID {
				continue
			}
			// Skip if already read
			if n.ReadAt != nil {
				continue
			}
			n.ReadAt = &now
			marked++
		}
	}

	return marked, nil
}

func (s *NotificationsMemoryStorage) MarkAllRead(ctx context.Context, profileID uuid.UUID) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids, ok := s.byProfile[profileID]
	if !ok {
		return 0, nil
	}

	marked := 0
	now := time.Now()

	for _, id := range ids {
		if n, ok := s.notifications[id]; ok {
			if n.ReadAt == nil {
				n.ReadAt = &now
				marked++
			}
		}
	}

	return marked, nil
}

// Helper functions

func makeUniqueKey(profileID uuid.UUID, kind string, sourceDate *time.Time) string {
	if sourceDate == nil {
		return profileID.String() + ":" + kind + ":null"
	}
	return profileID.String() + ":" + kind + ":" + sourceDate.Format("2006-01-02")
}

func sortByCreatedAtDesc(notifications []storage.Notification) {
	// Simple bubble sort (fine for small lists)
	for i := 0; i < len(notifications); i++ {
		for j := i + 1; j < len(notifications); j++ {
			if notifications[i].CreatedAt.Before(notifications[j].CreatedAt) {
				notifications[i], notifications[j] = notifications[j], notifications[i]
			}
		}
	}
}
