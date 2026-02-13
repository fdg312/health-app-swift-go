package memory

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
)

type SettingsMemoryStorage struct {
	mu       sync.RWMutex
	settings map[string]storage.Settings
}

func NewSettingsMemoryStorage() *SettingsMemoryStorage {
	return &SettingsMemoryStorage{
		settings: make(map[string]storage.Settings),
	}
}

func (s *SettingsMemoryStorage) GetSettings(ctx context.Context, ownerUserID string) (storage.Settings, bool, error) {
	_ = ctx
	key := strings.TrimSpace(ownerUserID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	row, ok := s.settings[key]
	if !ok {
		return storage.Settings{}, false, nil
	}
	return row, true, nil
}

func (s *SettingsMemoryStorage) UpsertSettings(ctx context.Context, ownerUserID string, in storage.Settings) (storage.Settings, error) {
	_ = ctx
	key := strings.TrimSpace(ownerUserID)

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	existing, ok := s.settings[key]
	if !ok {
		existing = storage.Settings{
			OwnerUserID: key,
			CreatedAt:   now,
		}
	}

	existing.OwnerUserID = key
	existing.TimeZone = in.TimeZone
	existing.QuietStartMinutes = in.QuietStartMinutes
	existing.QuietEndMinutes = in.QuietEndMinutes
	existing.NotificationsMaxPerDay = in.NotificationsMaxPerDay
	existing.MinSleepMinutes = in.MinSleepMinutes
	existing.MinSteps = in.MinSteps
	existing.MinActiveEnergyKcal = in.MinActiveEnergyKcal
	existing.MorningCheckinMinute = in.MorningCheckinMinute
	existing.EveningCheckinMinute = in.EveningCheckinMinute
	existing.VitaminsTimeMinute = in.VitaminsTimeMinute
	existing.UpdatedAt = now

	s.settings[key] = existing
	return existing, nil
}
