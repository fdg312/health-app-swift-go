package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/userctx"
	"github.com/google/uuid"
)

var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrInvalidDate     = errors.New("invalid date format")
	ErrInvalidRange    = errors.New("invalid date range")
	ErrInvalidStage    = errors.New("invalid sleep stage")
	ErrInvalidTime     = errors.New("invalid time range")
)

// Service содержит бизнес-логику метрик
type Service struct {
	profileStorage storage.Storage
	metricsStorage storage.MetricsStorage
}

// NewService создаёт новый сервис
func NewService(profileStorage storage.Storage, metricsStorage storage.MetricsStorage) *Service {
	return &Service{
		profileStorage: profileStorage,
		metricsStorage: metricsStorage,
	}
}

// SyncBatch обрабатывает батчевую синхронизацию
func (s *Service) SyncBatch(ctx context.Context, req SyncBatchRequest) (*SyncBatchResponse, error) {
	if err := s.ensureProfileAccess(ctx, req.ProfileID); err != nil {
		return nil, ErrProfileNotFound
	}

	resp := &SyncBatchResponse{
		Status: "ok",
	}

	// Сохраняем daily metrics
	for _, daily := range req.Daily {
		if err := s.validateDate(daily.Date); err != nil {
			return nil, err
		}

		payload, err := json.Marshal(daily)
		if err != nil {
			return nil, err
		}

		if err := s.metricsStorage.UpsertDailyMetric(ctx, req.ProfileID, daily.Date, payload); err != nil {
			return nil, err
		}

		resp.UpsertedDaily++
	}

	// Сохраняем hourly metrics
	for _, hourly := range req.Hourly {
		var steps, hrMin, hrMax, hrAvg *int
		if hourly.Steps != nil {
			steps = hourly.Steps
		}
		if hourly.HR != nil {
			hrMin = &hourly.HR.Min
			hrMax = &hourly.HR.Max
			hrAvg = &hourly.HR.Avg
		}

		if err := s.metricsStorage.UpsertHourlyMetric(ctx, req.ProfileID, hourly.Hour, steps, hrMin, hrMax, hrAvg); err != nil {
			return nil, err
		}

		resp.UpsertedHourly++
	}

	// Сохраняем sleep segments
	for _, seg := range req.Sessions.SleepSegments {
		if err := s.validateSleepStage(seg.Stage); err != nil {
			return nil, err
		}
		if err := s.validateTimeRange(seg.Start, seg.End); err != nil {
			return nil, err
		}

		if err := s.metricsStorage.InsertSleepSegment(ctx, req.ProfileID, seg.Start, seg.End, seg.Stage); err != nil {
			return nil, err
		}

		resp.InsertedSleepSegs++
	}

	// Сохраняем workouts
	for _, workout := range req.Sessions.Workouts {
		if err := s.validateTimeRange(workout.Start, workout.End); err != nil {
			return nil, err
		}

		if err := s.metricsStorage.InsertWorkout(ctx, req.ProfileID, workout.Start, workout.End, workout.Label, workout.CaloriesKcal); err != nil {
			return nil, err
		}

		resp.InsertedWorkouts++
	}

	return resp, nil
}

// GetDailyMetrics возвращает дневные метрики за период
func (s *Service) GetDailyMetrics(ctx context.Context, profileID uuid.UUID, from, to string) (*DailyMetricsResponse, error) {
	if err := s.ensureProfileAccess(ctx, profileID); err != nil {
		return nil, ErrProfileNotFound
	}

	if err := s.validateDate(from); err != nil {
		return nil, err
	}
	if err := s.validateDate(to); err != nil {
		return nil, err
	}
	if from > to {
		return nil, ErrInvalidRange
	}

	rows, err := s.metricsStorage.GetDailyMetrics(ctx, profileID, from, to)
	if err != nil {
		return nil, err
	}

	var dailyAggs []DailyAggregate
	for _, row := range rows {
		var agg DailyAggregate
		if err := json.Unmarshal(row.Payload, &agg); err != nil {
			continue // skip invalid
		}
		dailyAggs = append(dailyAggs, agg)
	}

	return &DailyMetricsResponse{Daily: dailyAggs}, nil
}

// GetHourlyMetrics возвращает часовые метрики за день
func (s *Service) GetHourlyMetrics(ctx context.Context, profileID uuid.UUID, date, metric string) (*HourlyMetricsResponse, error) {
	if err := s.ensureProfileAccess(ctx, profileID); err != nil {
		return nil, ErrProfileNotFound
	}

	if err := s.validateDate(date); err != nil {
		return nil, err
	}

	rows, err := s.metricsStorage.GetHourlyMetrics(ctx, profileID, date)
	if err != nil {
		return nil, err
	}

	var hourlyBuckets []HourlyBucket
	for _, row := range rows {
		bucket := HourlyBucket{
			Hour: row.Hour,
		}

		if metric == "steps" && row.Steps != nil {
			bucket.Steps = row.Steps
		}

		if metric == "hr" && row.HRMin != nil && row.HRMax != nil && row.HRAvg != nil {
			bucket.HR = &HRData{
				Min: *row.HRMin,
				Max: *row.HRMax,
				Avg: *row.HRAvg,
			}
		}

		// Добавляем только если есть данные для запрошенной метрики
		if (metric == "steps" && bucket.Steps != nil) || (metric == "hr" && bucket.HR != nil) {
			hourlyBuckets = append(hourlyBuckets, bucket)
		}
	}

	return &HourlyMetricsResponse{Hourly: hourlyBuckets}, nil
}

// Валидация

func (s *Service) validateDate(date string) error {
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ErrInvalidDate
	}
	return nil
}

func (s *Service) validateSleepStage(stage string) error {
	validStages := map[string]bool{
		"rem":   true,
		"deep":  true,
		"core":  true,
		"awake": true,
	}
	if !validStages[stage] {
		return ErrInvalidStage
	}
	return nil
}

func (s *Service) validateTimeRange(start, end time.Time) error {
	if start.After(end) || start.Equal(end) {
		return ErrInvalidTime
	}
	return nil
}

func (s *Service) ensureProfileAccess(ctx context.Context, profileID uuid.UUID) error {
	profile, err := s.profileStorage.GetProfile(ctx, profileID)
	if err != nil {
		return ErrProfileNotFound
	}

	if userID, ok := userctx.GetUserID(ctx); ok && strings.TrimSpace(userID) != "" && profile.OwnerUserID != userID {
		return ErrProfileNotFound
	}

	return nil
}
