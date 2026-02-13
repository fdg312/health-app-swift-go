package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

var (
	ErrDuplicateKey = errors.New("duplicate key")
)

// MetricsMemoryStorage — in-memory реализация MetricsStorage
type MetricsMemoryStorage struct {
	mu            sync.RWMutex
	dailyMetrics  map[string]storage.DailyMetricRow  // key: "profileID:date"
	hourlyMetrics map[string]storage.HourlyMetricRow // key: "profileID:hour"
	sleepSegments map[string]bool                     // key: "profileID:start:end:stage"
	workouts      map[string]bool                     // key: "profileID:start:end:label"
}

// NewMetricsStorage создаёт новый MetricsMemoryStorage
func NewMetricsStorage() *MetricsMemoryStorage {
	return &MetricsMemoryStorage{
		dailyMetrics:  make(map[string]storage.DailyMetricRow),
		hourlyMetrics: make(map[string]storage.HourlyMetricRow),
		sleepSegments: make(map[string]bool),
		workouts:      make(map[string]bool),
	}
}

func (m *MetricsMemoryStorage) UpsertDailyMetric(ctx context.Context, profileID uuid.UUID, date string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", profileID.String(), date)
	now := time.Now()

	existing, exists := m.dailyMetrics[key]
	if exists {
		existing.Payload = payload
		existing.UpdatedAt = now
		m.dailyMetrics[key] = existing
	} else {
		m.dailyMetrics[key] = storage.DailyMetricRow{
			ProfileID: profileID,
			Date:      date,
			Payload:   payload,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	return nil
}

func (m *MetricsMemoryStorage) GetDailyMetrics(ctx context.Context, profileID uuid.UUID, from, to string) ([]storage.DailyMetricRow, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []storage.DailyMetricRow
	for _, row := range m.dailyMetrics {
		if row.ProfileID == profileID && row.Date >= from && row.Date <= to {
			results = append(results, row)
		}
	}

	// Сортировка по дате
	// Простая сортировка для MVP
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Date > results[j].Date {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results, nil
}

func (m *MetricsMemoryStorage) UpsertHourlyMetric(ctx context.Context, profileID uuid.UUID, hour time.Time, steps *int, hrMin, hrMax, hrAvg *int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Округляем до начала часа
	hourTrunc := hour.Truncate(time.Hour)
	key := fmt.Sprintf("%s:%d", profileID.String(), hourTrunc.Unix())
	now := time.Now()

	existing, exists := m.hourlyMetrics[key]
	if exists {
		if steps != nil {
			existing.Steps = steps
		}
		if hrMin != nil {
			existing.HRMin = hrMin
		}
		if hrMax != nil {
			existing.HRMax = hrMax
		}
		if hrAvg != nil {
			existing.HRAvg = hrAvg
		}
		existing.UpdatedAt = now
		m.hourlyMetrics[key] = existing
	} else {
		m.hourlyMetrics[key] = storage.HourlyMetricRow{
			ProfileID: profileID,
			Hour:      hourTrunc,
			Steps:     steps,
			HRMin:     hrMin,
			HRMax:     hrMax,
			HRAvg:     hrAvg,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	return nil
}

func (m *MetricsMemoryStorage) GetHourlyMetrics(ctx context.Context, profileID uuid.UUID, date string) ([]storage.HourlyMetricRow, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []storage.HourlyMetricRow
	for _, row := range m.hourlyMetrics {
		if row.ProfileID == profileID {
			rowDate := row.Hour.Format("2006-01-02")
			if rowDate == date {
				results = append(results, row)
			}
		}
	}

	// Сортировка по часу
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Hour.After(results[j].Hour) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results, nil
}

func (m *MetricsMemoryStorage) InsertSleepSegment(ctx context.Context, profileID uuid.UUID, start, end time.Time, stage string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%d:%d:%s", profileID.String(), start.Unix(), end.Unix(), stage)
	if m.sleepSegments[key] {
		return nil // ignore duplicate
	}

	m.sleepSegments[key] = true
	return nil
}

func (m *MetricsMemoryStorage) InsertWorkout(ctx context.Context, profileID uuid.UUID, start, end time.Time, label string, caloriesKcal *int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%d:%d:%s", profileID.String(), start.Unix(), end.Unix(), label)
	if m.workouts[key] {
		return nil // ignore duplicate
	}

	m.workouts[key] = true
	return nil
}
