package memory

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

var (
	ErrSourceNotFound = errors.New("source not found")
	ErrBlobNotFound   = errors.New("blob not found")
)

// SourcesMemoryStorage — in-memory реализация SourcesStorage
type SourcesMemoryStorage struct {
	mu      sync.RWMutex
	sources map[uuid.UUID]storage.Source
	blobs   map[uuid.UUID]blobData // For image sources in memory mode
}

type blobData struct {
	Data        []byte
	ContentType string
}

// NewSourcesMemoryStorage создаёт новый SourcesMemoryStorage
func NewSourcesMemoryStorage() *SourcesMemoryStorage {
	return &SourcesMemoryStorage{
		sources: make(map[uuid.UUID]storage.Source),
		blobs:   make(map[uuid.UUID]blobData),
	}
}

func (s *SourcesMemoryStorage) CreateSource(ctx context.Context, source *storage.Source) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if source.ID == uuid.Nil {
		source.ID = uuid.New()
	}

	now := time.Now()
	source.CreatedAt = now
	source.UpdatedAt = now

	s.sources[source.ID] = *source

	return nil
}

func (s *SourcesMemoryStorage) GetSource(ctx context.Context, id uuid.UUID) (*storage.Source, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	src, ok := s.sources[id]
	if !ok {
		return nil, ErrSourceNotFound
	}

	return &src, nil
}

func (s *SourcesMemoryStorage) ListSources(ctx context.Context, profileID uuid.UUID, query string, checkinID *uuid.UUID, limit, offset int) ([]storage.Source, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Filter by profile and optional checkin
	var filtered []storage.Source
	for _, src := range s.sources {
		if src.ProfileID != profileID {
			continue
		}

		// Filter by checkin if specified
		if checkinID != nil {
			if src.CheckinID == nil || *src.CheckinID != *checkinID {
				continue
			}
		}

		// Simple search by query (case-insensitive substring match)
		if query != "" {
			lowerQuery := strings.ToLower(query)
			matched := false

			if src.Title != nil && strings.Contains(strings.ToLower(*src.Title), lowerQuery) {
				matched = true
			}
			if src.Text != nil && strings.Contains(strings.ToLower(*src.Text), lowerQuery) {
				matched = true
			}
			if src.URL != nil && strings.Contains(strings.ToLower(*src.URL), lowerQuery) {
				matched = true
			}

			if !matched {
				continue
			}
		}

		filtered = append(filtered, src)
	}

	// Sort by created_at descending (most recent first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	// Apply pagination
	start := offset
	if start > len(filtered) {
		return []storage.Source{}, nil
	}

	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], nil
}

func (s *SourcesMemoryStorage) DeleteSource(ctx context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sources[id]; !ok {
		return ErrSourceNotFound
	}

	delete(s.sources, id)
	delete(s.blobs, id) // Also delete blob if exists

	return nil
}

func (s *SourcesMemoryStorage) GetSourceBlob(ctx context.Context, sourceID uuid.UUID) ([]byte, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	blob, ok := s.blobs[sourceID]
	if !ok {
		return nil, "", ErrBlobNotFound
	}

	return blob.Data, blob.ContentType, nil
}

func (s *SourcesMemoryStorage) PutSourceBlob(ctx context.Context, sourceID uuid.UUID, data []byte, contentType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.blobs[sourceID] = blobData{
		Data:        data,
		ContentType: contentType,
	}

	return nil
}
