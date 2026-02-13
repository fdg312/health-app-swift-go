package memory

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

// SupplementSchedulesMemoryStorage — in-memory storage for supplement schedules.
type SupplementSchedulesMemoryStorage struct {
	mu             sync.RWMutex
	schedules      map[uuid.UUID]*storage.SupplementSchedule
	byOwnerProfile map[string][]uuid.UUID // owner:profile -> schedule_ids
	unique         map[string]uuid.UUID   // owner:profile:supplement:time -> schedule_id
}

func NewSupplementSchedulesMemoryStorage() *SupplementSchedulesMemoryStorage {
	return &SupplementSchedulesMemoryStorage{
		schedules:      make(map[uuid.UUID]*storage.SupplementSchedule),
		byOwnerProfile: make(map[string][]uuid.UUID),
		unique:         make(map[string]uuid.UUID),
	}
}

func (s *SupplementSchedulesMemoryStorage) ListSchedules(ctx context.Context, ownerUserID string, profileID uuid.UUID) ([]storage.SupplementSchedule, error) {
	_ = ctx

	ownerUserID = strings.TrimSpace(ownerUserID)
	ownerProfileKey := ownerProfileKey(ownerUserID, profileID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.byOwnerProfile[ownerProfileKey]
	result := make([]storage.SupplementSchedule, 0, len(ids))
	for _, id := range ids {
		if row, ok := s.schedules[id]; ok {
			result = append(result, *row)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].TimeMinutes == result[j].TimeMinutes {
			return result[i].CreatedAt.Before(result[j].CreatedAt)
		}
		return result[i].TimeMinutes < result[j].TimeMinutes
	})

	return result, nil
}

func (s *SupplementSchedulesMemoryStorage) UpsertSchedule(ctx context.Context, ownerUserID string, profileID uuid.UUID, item storage.ScheduleUpsert) (storage.SupplementSchedule, error) {
	_ = ctx

	ownerUserID = strings.TrimSpace(ownerUserID)
	ownerProfileKey := ownerProfileKey(ownerUserID, profileID)
	uniqueKey := scheduleUniqueKey(ownerUserID, profileID, item.SupplementID, item.TimeMinutes)
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	if id, ok := s.unique[uniqueKey]; ok {
		if existing, ok := s.schedules[id]; ok {
			existing.DaysMask = item.DaysMask
			existing.IsEnabled = item.IsEnabled
			existing.UpdatedAt = now
			return *existing, nil
		}
	}

	row := storage.SupplementSchedule{
		ID:           uuid.New(),
		OwnerUserID:  ownerUserID,
		ProfileID:    profileID,
		SupplementID: item.SupplementID,
		TimeMinutes:  item.TimeMinutes,
		DaysMask:     item.DaysMask,
		IsEnabled:    item.IsEnabled,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	s.schedules[row.ID] = &row
	s.unique[uniqueKey] = row.ID
	s.byOwnerProfile[ownerProfileKey] = append(s.byOwnerProfile[ownerProfileKey], row.ID)

	return row, nil
}

func (s *SupplementSchedulesMemoryStorage) DeleteSchedule(ctx context.Context, ownerUserID string, scheduleID uuid.UUID) error {
	_ = ctx

	ownerUserID = strings.TrimSpace(ownerUserID)

	s.mu.Lock()
	defer s.mu.Unlock()

	row, ok := s.schedules[scheduleID]
	if !ok || row.OwnerUserID != ownerUserID {
		return ErrNotFound
	}

	ownerProfileKey := ownerProfileKey(ownerUserID, row.ProfileID)
	uniqueKey := scheduleUniqueKey(ownerUserID, row.ProfileID, row.SupplementID, row.TimeMinutes)

	delete(s.schedules, scheduleID)
	delete(s.unique, uniqueKey)

	ids := s.byOwnerProfile[ownerProfileKey]
	filtered := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id != scheduleID {
			filtered = append(filtered, id)
		}
	}
	if len(filtered) == 0 {
		delete(s.byOwnerProfile, ownerProfileKey)
	} else {
		s.byOwnerProfile[ownerProfileKey] = filtered
	}

	return nil
}

func (s *SupplementSchedulesMemoryStorage) ReplaceAll(ctx context.Context, ownerUserID string, profileID uuid.UUID, items []storage.ScheduleUpsert) ([]storage.SupplementSchedule, error) {
	_ = ctx

	ownerUserID = strings.TrimSpace(ownerUserID)
	ownerProfileKey := ownerProfileKey(ownerUserID, profileID)
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing rows for owner/profile first.
	oldIDs := s.byOwnerProfile[ownerProfileKey]
	for _, id := range oldIDs {
		row, ok := s.schedules[id]
		if !ok {
			continue
		}
		delete(s.unique, scheduleUniqueKey(ownerUserID, profileID, row.SupplementID, row.TimeMinutes))
		delete(s.schedules, id)
	}
	delete(s.byOwnerProfile, ownerProfileKey)

	saved := make([]storage.SupplementSchedule, 0, len(items))
	for _, item := range items {
		uniqueKey := scheduleUniqueKey(ownerUserID, profileID, item.SupplementID, item.TimeMinutes)
		if existingID, ok := s.unique[uniqueKey]; ok {
			if existing, ok := s.schedules[existingID]; ok {
				existing.DaysMask = item.DaysMask
				existing.IsEnabled = item.IsEnabled
				existing.UpdatedAt = now
				continue
			}
		}

		row := storage.SupplementSchedule{
			ID:           uuid.New(),
			OwnerUserID:  ownerUserID,
			ProfileID:    profileID,
			SupplementID: item.SupplementID,
			TimeMinutes:  item.TimeMinutes,
			DaysMask:     item.DaysMask,
			IsEnabled:    item.IsEnabled,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		s.schedules[row.ID] = &row
		s.unique[uniqueKey] = row.ID
		s.byOwnerProfile[ownerProfileKey] = append(s.byOwnerProfile[ownerProfileKey], row.ID)
		saved = append(saved, row)
	}

	sort.Slice(saved, func(i, j int) bool {
		if saved[i].TimeMinutes == saved[j].TimeMinutes {
			return saved[i].CreatedAt.Before(saved[j].CreatedAt)
		}
		return saved[i].TimeMinutes < saved[j].TimeMinutes
	})

	return saved, nil
}

func ownerProfileKey(ownerUserID string, profileID uuid.UUID) string {
	return ownerUserID + ":" + profileID.String()
}

func scheduleUniqueKey(ownerUserID string, profileID, supplementID uuid.UUID, timeMinutes int) string {
	return ownerUserID + ":" + profileID.String() + ":" + supplementID.String() + ":" + strconv.Itoa(timeMinutes)
}
