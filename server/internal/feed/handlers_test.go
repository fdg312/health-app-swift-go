package feed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

// mockMetricsStorage implements MetricsStorage for testing
type mockMetricsStorage struct {
	metrics map[string][]byte // key: "profileID:date"
}

func newMockMetricsStorage() *mockMetricsStorage {
	return &mockMetricsStorage{
		metrics: make(map[string][]byte),
	}
}

func (m *mockMetricsStorage) GetDailyMetrics(profileID uuid.UUID, from, to string) ([]DailyMetricRow, error) {
	key := profileID.String() + ":" + from
	if payload, exists := m.metrics[key]; exists {
		return []DailyMetricRow{
			{
				ProfileID: profileID,
				Date:      from,
				Payload:   payload,
			},
		}, nil
	}
	return []DailyMetricRow{}, nil
}

func (m *mockMetricsStorage) AddMetric(profileID uuid.UUID, date string, payload []byte) {
	key := profileID.String() + ":" + date
	m.metrics[key] = payload
}

// mockCheckinsStorage implements CheckinsStorage for testing
type mockCheckinsStorage struct {
	checkins []Checkin
}

func newMockCheckinsStorage() *mockCheckinsStorage {
	return &mockCheckinsStorage{
		checkins: []Checkin{},
	}
}

func (m *mockCheckinsStorage) ListCheckins(profileID uuid.UUID, from, to string) ([]Checkin, error) {
	var result []Checkin
	for _, c := range m.checkins {
		if c.ProfileID == profileID && c.Date >= from && c.Date <= to {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *mockCheckinsStorage) AddCheckin(c Checkin) {
	m.checkins = append(m.checkins, c)
}

// mockProfileStorage implements ProfileStorage for testing
type mockProfileStorage struct {
	profiles map[uuid.UUID]storage.Profile
}

func newMockProfileStorage() *mockProfileStorage {
	return &mockProfileStorage{
		profiles: make(map[uuid.UUID]storage.Profile),
	}
}

func (m *mockProfileStorage) GetProfile(ctx context.Context, id uuid.UUID) (*storage.Profile, error) {
	profile, exists := m.profiles[id]
	if !exists {
		return nil, ErrProfileNotFound
	}
	return &profile, nil
}

func (m *mockProfileStorage) AddProfile(id uuid.UUID) {
	m.profiles[id] = storage.Profile{
		ID:          id,
		OwnerUserID: "default",
		Type:        "owner",
	}
}

func TestHandleGetDay_HappyPath(t *testing.T) {
	// Setup
	metricsStorage := newMockMetricsStorage()
	checkinsStorage := newMockCheckinsStorage()
	profileStorage := newMockProfileStorage()

	profileID := uuid.New()
	profileStorage.AddProfile(profileID)

	// Add daily metrics
	dailyPayload := []byte(`{
		"activity": {"steps": 10000},
		"body": {"weight_kg_last": 70.5},
		"heart": {"resting_hr_bpm": 60}
	}`)
	metricsStorage.AddMetric(profileID, "2026-02-12", dailyPayload)

	// Add checkins
	checkinsStorage.AddCheckin(Checkin{
		ID:        uuid.New(),
		ProfileID: profileID,
		Date:      "2026-02-12",
		Type:      "morning",
		Score:     4,
		Tags:      []string{},
		Note:      "good morning",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	checkinsStorage.AddCheckin(Checkin{
		ID:        uuid.New(),
		ProfileID: profileID,
		Date:      "2026-02-12",
		Type:      "evening",
		Score:     3,
		Tags:      []string{"tired"},
		Note:      "tired evening",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	service := NewService(metricsStorage, checkinsStorage, profileStorage, nil)

	// Test
	req := httptest.NewRequest("GET", "/v1/feed/day?profile_id="+profileID.String()+"&date=2026-02-12", nil)
	w := httptest.NewRecorder()

	HandleGetDay(service)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp FeedDayResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Date != "2026-02-12" {
		t.Errorf("expected date 2026-02-12, got %s", resp.Date)
	}

	if resp.Daily == nil {
		t.Error("expected daily metrics to be present")
	}

	if resp.Checkins == nil || resp.Checkins.Morning == nil || resp.Checkins.Evening == nil {
		t.Error("expected both morning and evening checkins")
	}

	if len(resp.MissingFields) != 0 {
		t.Errorf("expected no missing fields, got %v", resp.MissingFields)
	}
}

func TestHandleGetDay_PartialData_NoCheckins(t *testing.T) {
	// Setup
	metricsStorage := newMockMetricsStorage()
	checkinsStorage := newMockCheckinsStorage()
	profileStorage := newMockProfileStorage()

	profileID := uuid.New()
	profileStorage.AddProfile(profileID)

	// Add daily metrics only
	dailyPayload := []byte(`{
		"activity": {"steps": 5000},
		"body": {"weight_kg_last": 70.5},
		"heart": {"resting_hr_bpm": 62}
	}`)
	metricsStorage.AddMetric(profileID, "2026-02-12", dailyPayload)

	service := NewService(metricsStorage, checkinsStorage, profileStorage, nil)

	// Test
	req := httptest.NewRequest("GET", "/v1/feed/day?profile_id="+profileID.String()+"&date=2026-02-12", nil)
	w := httptest.NewRecorder()

	HandleGetDay(service)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp FeedDayResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Daily == nil {
		t.Error("expected daily metrics to be present")
	}

	if len(resp.MissingFields) != 2 {
		t.Errorf("expected 2 missing fields (morning_checkin, evening_checkin), got %d: %v", len(resp.MissingFields), resp.MissingFields)
	}
}

func TestHandleGetDay_PartialData_NoDaily(t *testing.T) {
	// Setup
	metricsStorage := newMockMetricsStorage()
	checkinsStorage := newMockCheckinsStorage()
	profileStorage := newMockProfileStorage()

	profileID := uuid.New()
	profileStorage.AddProfile(profileID)

	// Add checkins only
	checkinsStorage.AddCheckin(Checkin{
		ID:        uuid.New(),
		ProfileID: profileID,
		Date:      "2026-02-12",
		Type:      "morning",
		Score:     5,
		Tags:      []string{},
		Note:      "",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	service := NewService(metricsStorage, checkinsStorage, profileStorage, nil)

	// Test
	req := httptest.NewRequest("GET", "/v1/feed/day?profile_id="+profileID.String()+"&date=2026-02-12", nil)
	w := httptest.NewRecorder()

	HandleGetDay(service)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp FeedDayResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Daily != nil {
		t.Error("expected no daily metrics")
	}

	// Should have: daily, evening_checkin
	if len(resp.MissingFields) != 2 {
		t.Errorf("expected 2 missing fields, got %d: %v", len(resp.MissingFields), resp.MissingFields)
	}
}

func TestHandleGetDay_InvalidDate(t *testing.T) {
	// Setup
	metricsStorage := newMockMetricsStorage()
	checkinsStorage := newMockCheckinsStorage()
	profileStorage := newMockProfileStorage()

	profileID := uuid.New()
	profileStorage.AddProfile(profileID)

	service := NewService(metricsStorage, checkinsStorage, profileStorage, nil)

	// Test with invalid date
	req := httptest.NewRequest("GET", "/v1/feed/day?profile_id="+profileID.String()+"&date=invalid-date", nil)
	w := httptest.NewRecorder()

	HandleGetDay(service)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Error.Code != "invalid_date" {
		t.Errorf("expected error code invalid_date, got %s", resp.Error.Code)
	}
}

func TestHandleGetDay_ProfileNotFound(t *testing.T) {
	// Setup
	metricsStorage := newMockMetricsStorage()
	checkinsStorage := newMockCheckinsStorage()
	profileStorage := newMockProfileStorage()

	service := NewService(metricsStorage, checkinsStorage, profileStorage, nil)

	// Test with non-existent profile
	randomID := uuid.New()
	req := httptest.NewRequest("GET", "/v1/feed/day?profile_id="+randomID.String()+"&date=2026-02-12", nil)
	w := httptest.NewRecorder()

	HandleGetDay(service)(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var resp ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Error.Code != "profile_not_found" {
		t.Errorf("expected error code profile_not_found, got %s", resp.Error.Code)
	}
}
