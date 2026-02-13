package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fdg312/health-hub/internal/storage/memory"
	"github.com/google/uuid"
)

func TestHandleSyncBatch(t *testing.T) {
	store := memory.New()
	service := NewService(store, store)
	handler := NewHandler(service)

	// Получаем owner profile
	profiles, _ := store.ListProfiles(context.Background())
	ownerID := profiles[0].ID

	// Создаём sync batch request
	now := time.Now().Truncate(time.Hour)
	reqBody := SyncBatchRequest{
		ProfileID: ownerID,
		Daily: []DailyAggregate{
			{
				Date: "2026-02-12",
				Sleep: &SleepDaily{
					TotalMinutes: 480,
					Stages: &SleepStages{
						Rem:   120,
						Deep:  180,
						Core:  150,
						Awake: 30,
					},
				},
				Activity: &ActivityDaily{
					Steps:           10000,
					ActiveEnergyKcal: 500,
					ExerciseMin:     30,
					StandHours:      12,
					DistanceKm:      7.5,
				},
			},
		},
		Hourly: []HourlyBucket{
			{
				Hour:  now,
				Steps: intPtr(1500),
				HR: &HRData{
					Min: 60,
					Max: 85,
					Avg: 72,
				},
			},
		},
		Sessions: Sessions{
			SleepSegments: []SleepSegment{
				{
					Start: now.Add(-8 * time.Hour),
					End:   now.Add(-6 * time.Hour),
					Stage: "deep",
				},
			},
			Workouts: []WorkoutSession{
				{
					Start:        now.Add(-2 * time.Hour),
					End:          now.Add(-1 * time.Hour),
					Label:        "run",
					CaloriesKcal: intPtr(300),
				},
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/sync/batch", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleSyncBatch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp SyncBatchResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status=ok, got %s", resp.Status)
	}

	if resp.UpsertedDaily != 1 {
		t.Errorf("expected upserted_daily=1, got %d", resp.UpsertedDaily)
	}

	if resp.UpsertedHourly != 1 {
		t.Errorf("expected upserted_hourly=1, got %d", resp.UpsertedHourly)
	}

	if resp.InsertedSleepSegs != 1 {
		t.Errorf("expected inserted_sleep_segments=1, got %d", resp.InsertedSleepSegs)
	}

	if resp.InsertedWorkouts != 1 {
		t.Errorf("expected inserted_workouts=1, got %d", resp.InsertedWorkouts)
	}
}

func TestHandleSyncBatchProfileNotFound(t *testing.T) {
	store := memory.New()
	service := NewService(store, store)
	handler := NewHandler(service)

	reqBody := SyncBatchRequest{
		ProfileID: uuid.New(), // несуществующий профиль
		Daily:     []DailyAggregate{{Date: "2026-02-12"}},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/sync/batch", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleSyncBatch(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error.Code != "profile_not_found" {
		t.Errorf("expected error code 'profile_not_found', got %s", resp.Error.Code)
	}
}

func TestHandleGetDailyMetrics(t *testing.T) {
	store := memory.New()
	service := NewService(store, store)
	handler := NewHandler(service)

	// Получаем owner profile
	profiles, _ := store.ListProfiles(context.Background())
	ownerID := profiles[0].ID

	// Сначала отправим данные
	reqBody := SyncBatchRequest{
		ProfileID: ownerID,
		Daily: []DailyAggregate{
			{Date: "2026-02-10", Activity: &ActivityDaily{Steps: 8000}},
			{Date: "2026-02-11", Activity: &ActivityDaily{Steps: 9000}},
			{Date: "2026-02-12", Activity: &ActivityDaily{Steps: 10000}},
		},
	}
	service.SyncBatch(context.Background(), reqBody)

	// Теперь запрашиваем метрики
	req := httptest.NewRequest(http.MethodGet,
		"/v1/metrics/daily?profile_id="+ownerID.String()+"&from=2026-02-10&to=2026-02-12", nil)
	w := httptest.NewRecorder()

	handler.HandleGetDailyMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp DailyMetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Daily) != 3 {
		t.Errorf("expected 3 daily metrics, got %d", len(resp.Daily))
	}
}

func TestHandleGetHourlyMetricsSteps(t *testing.T) {
	store := memory.New()
	service := NewService(store, store)
	handler := NewHandler(service)

	// Получаем owner profile
	profiles, _ := store.ListProfiles(context.Background())
	ownerID := profiles[0].ID

	// Отправим часовые данные
	now := time.Date(2026, 2, 12, 10, 0, 0, 0, time.UTC)
	reqBody := SyncBatchRequest{
		ProfileID: ownerID,
		Hourly: []HourlyBucket{
			{Hour: now, Steps: intPtr(1500)},
			{Hour: now.Add(time.Hour), Steps: intPtr(2000)},
			{Hour: now.Add(2 * time.Hour), Steps: intPtr(1800)},
		},
	}
	service.SyncBatch(context.Background(), reqBody)

	// Запрашиваем часовые метрики по шагам
	req := httptest.NewRequest(http.MethodGet,
		"/v1/metrics/hourly?profile_id="+ownerID.String()+"&date=2026-02-12&metric=steps", nil)
	w := httptest.NewRecorder()

	handler.HandleGetHourlyMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp HourlyMetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Hourly) != 3 {
		t.Errorf("expected 3 hourly metrics, got %d", len(resp.Hourly))
	}
}

func TestHandleGetHourlyMetricsHR(t *testing.T) {
	store := memory.New()
	service := NewService(store, store)
	handler := NewHandler(service)

	// Получаем owner profile
	profiles, _ := store.ListProfiles(context.Background())
	ownerID := profiles[0].ID

	// Отправим часовые данные с HR
	now := time.Date(2026, 2, 12, 14, 0, 0, 0, time.UTC)
	reqBody := SyncBatchRequest{
		ProfileID: ownerID,
		Hourly: []HourlyBucket{
			{
				Hour: now,
				HR: &HRData{
					Min: 60,
					Max: 90,
					Avg: 75,
				},
			},
			{
				Hour: now.Add(time.Hour),
				HR: &HRData{
					Min: 65,
					Max: 95,
					Avg: 80,
				},
			},
		},
	}
	service.SyncBatch(context.Background(), reqBody)

	// Запрашиваем часовые метрики по HR
	req := httptest.NewRequest(http.MethodGet,
		"/v1/metrics/hourly?profile_id="+ownerID.String()+"&date=2026-02-12&metric=hr", nil)
	w := httptest.NewRecorder()

	handler.HandleGetHourlyMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp HourlyMetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Hourly) != 2 {
		t.Errorf("expected 2 hourly metrics, got %d", len(resp.Hourly))
	}

	if resp.Hourly[0].HR == nil {
		t.Error("expected HR data in first bucket")
	}
}

func intPtr(v int) *int {
	return &v
}
