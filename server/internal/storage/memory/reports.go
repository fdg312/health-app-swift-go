package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

// ReportsMemoryStorage — in-memory storage для отчётов
type ReportsMemoryStorage struct {
	mu      sync.RWMutex
	reports map[uuid.UUID]*storage.ReportMeta
}

// NewReportsMemoryStorage создаёт новое in-memory хранилище
func NewReportsMemoryStorage() *ReportsMemoryStorage {
	return &ReportsMemoryStorage{
		reports: make(map[uuid.UUID]*storage.ReportMeta),
	}
}

// CreateReport создаёт новый отчёт
func (s *ReportsMemoryStorage) CreateReport(ctx context.Context, report *storage.ReportMeta) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if report.ID == uuid.Nil {
		report.ID = uuid.New()
	}

	now := time.Now()
	report.CreatedAt = now
	report.UpdatedAt = now

	s.reports[report.ID] = report
	return nil
}

// GetReport возвращает отчёт по ID
func (s *ReportsMemoryStorage) GetReport(ctx context.Context, id uuid.UUID) (*storage.ReportMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	report, exists := s.reports[id]
	if !exists {
		return nil, fmt.Errorf("report not found")
	}

	return report, nil
}

// ListReports возвращает список отчётов с пагинацией
func (s *ReportsMemoryStorage) ListReports(ctx context.Context, profileID uuid.UUID, limit, offset int) ([]storage.ReportMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Собираем отчёты для профиля
	var filtered []storage.ReportMeta
	for _, r := range s.reports {
		if r.ProfileID == profileID {
			filtered = append(filtered, *r)
		}
	}

	// Сортируем по created_at DESC
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	// Применяем пагинацию
	start := offset
	if start > len(filtered) {
		return []storage.ReportMeta{}, nil
	}

	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], nil
}

// DeleteReport удаляет отчёт
func (s *ReportsMemoryStorage) DeleteReport(ctx context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.reports[id]; !exists {
		return fmt.Errorf("report not found")
	}

	delete(s.reports, id)
	return nil
}
