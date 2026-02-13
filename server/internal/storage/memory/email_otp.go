package memory

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

// EmailOTPMemoryStorage stores OTP codes in memory for local/dev usage.
type EmailOTPMemoryStorage struct {
	mu      sync.RWMutex
	records map[uuid.UUID]storage.EmailOTP
}

func NewEmailOTPMemoryStorage() *EmailOTPMemoryStorage {
	return &EmailOTPMemoryStorage{
		records: make(map[uuid.UUID]storage.EmailOTP),
	}
}

func (s *EmailOTPMemoryStorage) CreateOrReplace(ctx context.Context, email, codeHash string, expiresAt, now time.Time, maxAttempts int) (uuid.UUID, error) {
	_ = ctx

	email = strings.TrimSpace(strings.ToLower(email))
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Keep history, but replace currently active OTP entries for the same email.
	for id, row := range s.records {
		if row.Email == email && row.ExpiresAt.After(now) {
			delete(s.records, id)
		}
	}

	otpID := uuid.New()
	s.records[otpID] = storage.EmailOTP{
		ID:          otpID,
		Email:       email,
		CodeHash:    codeHash,
		CreatedAt:   now,
		ExpiresAt:   expiresAt,
		Attempts:    0,
		MaxAttempts: maxAttempts,
		LastSentAt:  now,
		SendCount:   1,
	}
	return otpID, nil
}

func (s *EmailOTPMemoryStorage) GetLatestActive(ctx context.Context, email string, now time.Time) (*storage.EmailOTP, error) {
	_ = ctx
	email = strings.TrimSpace(strings.ToLower(email))

	s.mu.RLock()
	defer s.mu.RUnlock()

	var latest *storage.EmailOTP
	for _, row := range s.records {
		if row.Email != email || !row.ExpiresAt.After(now) {
			continue
		}
		if latest == nil || row.CreatedAt.After(latest.CreatedAt) {
			copyRow := row
			latest = &copyRow
		}
	}

	return latest, nil
}

func (s *EmailOTPMemoryStorage) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	_ = ctx

	s.mu.Lock()
	defer s.mu.Unlock()

	row, ok := s.records[id]
	if !ok {
		return nil
	}
	row.Attempts++
	s.records[id] = row
	return nil
}

func (s *EmailOTPMemoryStorage) MarkUsedOrDelete(ctx context.Context, id uuid.UUID) error {
	_ = ctx

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, id)
	return nil
}

func (s *EmailOTPMemoryStorage) UpdateResendMeta(ctx context.Context, id uuid.UUID, lastSentAt time.Time, sendCount int) error {
	_ = ctx

	s.mu.Lock()
	defer s.mu.Unlock()

	row, ok := s.records[id]
	if !ok {
		return nil
	}
	row.LastSentAt = lastSentAt
	row.SendCount = sendCount
	s.records[id] = row
	return nil
}
