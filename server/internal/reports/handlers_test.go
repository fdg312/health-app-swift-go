package reports

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/storage/memory"
	"github.com/google/uuid"
)

// Mock implementations
type mockCheckinsStorage struct{}

func (m *mockCheckinsStorage) ListCheckins(ctx context.Context, profileID uuid.UUID, from, to string) ([]Checkin, error) {
	return []Checkin{
		{
			ID:        uuid.New(),
			ProfileID: profileID,
			Date:      from,
			Type:      "morning",
			Score:     4,
			Tags:      []string{"энергия"},
			Note:      "Хорошее утро",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}, nil
}

type mockProfileStorage struct {
	profiles map[uuid.UUID]*storage.Profile
}

func newMockProfileStorage() *mockProfileStorage {
	return &mockProfileStorage{
		profiles: make(map[uuid.UUID]*storage.Profile),
	}
}

func (m *mockProfileStorage) GetProfile(ctx context.Context, id uuid.UUID) (*storage.Profile, error) {
	if prof, ok := m.profiles[id]; ok {
		return prof, nil
	}
	return nil, fmt.Errorf("profile not found")
}

func (m *mockProfileStorage) addProfile(id uuid.UUID) {
	m.profiles[id] = &storage.Profile{
		ID:          id,
		OwnerUserID: "default",
		Type:        "owner",
		Name:        "Test User",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func setupTestService() (*Service, uuid.UUID) {
	reportsStorage := memory.NewReportsMemoryStorage()
	metricsStorage := memory.NewMetricsStorage()
	checkinsStorage := &mockCheckinsStorage{}
	profileStorage := newMockProfileStorage()

	profileID := uuid.New()
	profileStorage.addProfile(profileID)

	// Add test metrics
	metricsStorage.UpsertDailyMetric(context.Background(), profileID, "2026-02-10", []byte(`{"activity":{"steps":10000}}`))

	service := NewService(
		reportsStorage,
		metricsStorage,
		checkinsStorage,
		profileStorage,
		nil,   // No S3, local mode
		90,    // max range days
		900,   // presign TTL
		"",    // publicBaseURL
		false, // preferPublicURL
	)

	return service, profileID
}

func TestHandleCreate_CSV_Success(t *testing.T) {
	service, profileID := setupTestService()
	handler := NewHandlers(service)

	reqBody := CreateReportRequest{
		ProfileID: profileID,
		From:      "2026-02-01",
		To:        "2026-02-15",
		Format:    FormatCSV,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/reports", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleCreate(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var resp ReportDTO
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Format != FormatCSV {
		t.Errorf("expected format csv, got %s", resp.Format)
	}

	if resp.DownloadURL == "" {
		t.Error("expected download URL")
	}
}

func TestHandleCreate_PDF_Success(t *testing.T) {
	// Skip custom font in tests to avoid path/embedding issues
	t.Setenv("SKIP_CUSTOM_FONT", "1")

	service, profileID := setupTestService()
	handler := NewHandlers(service)

	reqBody := CreateReportRequest{
		ProfileID: profileID,
		From:      "2026-02-01",
		To:        "2026-02-15",
		Format:    FormatPDF,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/reports", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleCreate(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp ReportDTO
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Format != FormatPDF {
		t.Errorf("expected format pdf, got %s", resp.Format)
	}
}

func TestHandleCreate_InvalidRange(t *testing.T) {
	service, profileID := setupTestService()
	handler := NewHandlers(service)

	reqBody := CreateReportRequest{
		ProfileID: profileID,
		From:      "2026-01-01",
		To:        "2026-06-01", // 5 months > 90 days
		Format:    FormatCSV,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/reports", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleCreate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var errResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&errResp)
	errorData := errResp["error"].(map[string]interface{})
	if errorData["code"] != "range_too_large" {
		t.Errorf("expected error code range_too_large, got %s", errorData["code"])
	}
}

func TestHandleCreate_ProfileNotFound(t *testing.T) {
	service, _ := setupTestService()
	handler := NewHandlers(service)

	reqBody := CreateReportRequest{
		ProfileID: uuid.New(), // Non-existent profile
		From:      "2026-02-01",
		To:        "2026-02-15",
		Format:    FormatCSV,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/reports", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleCreate(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandleList(t *testing.T) {
	service, profileID := setupTestService()
	handler := NewHandlers(service)

	// Create a report first
	service.CreateReport(context.Background(), CreateReportRequest{
		ProfileID: profileID,
		From:      "2026-02-01",
		To:        "2026-02-15",
		Format:    FormatCSV,
	})

	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/reports?profile_id=%s", profileID.String()), nil)
	w := httptest.NewRecorder()

	handler.HandleList(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp ReportsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Reports) != 1 {
		t.Errorf("expected 1 report, got %d", len(resp.Reports))
	}
}

func TestHandleDownload_LocalMode(t *testing.T) {
	service, profileID := setupTestService()
	handler := NewHandlers(service)

	// Create a CSV report
	report, err := service.CreateReport(context.Background(), CreateReportRequest{
		ProfileID: profileID,
		From:      "2026-02-01",
		To:        "2026-02-15",
		Format:    FormatCSV,
	})
	if err != nil {
		t.Fatalf("failed to create report: %v", err)
	}

	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/reports/%s/download", report.ID.String()), nil)
	req.SetPathValue("id", report.ID.String())
	w := httptest.NewRecorder()

	handler.HandleDownload(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "text/csv" {
		t.Errorf("expected content type text/csv, got %s", w.Header().Get("Content-Type"))
	}

	if w.Body.Len() == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestHandleDelete(t *testing.T) {
	service, profileID := setupTestService()
	handler := NewHandlers(service)

	// Create a report
	report, err := service.CreateReport(context.Background(), CreateReportRequest{
		ProfileID: profileID,
		From:      "2026-02-01",
		To:        "2026-02-15",
		Format:    FormatCSV,
	})
	if err != nil {
		t.Fatalf("failed to create report: %v", err)
	}

	// Delete it
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/v1/reports/%s", report.ID.String()), nil)
	req.SetPathValue("id", report.ID.String())
	w := httptest.NewRecorder()

	handler.HandleDelete(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}

	// Verify it's deleted
	_, err = service.GetReport(context.Background(), report.ID)
	if err == nil {
		t.Error("expected report to be deleted")
	}
}

func TestHandleDelete_NotFound(t *testing.T) {
	service, _ := setupTestService()
	handler := NewHandlers(service)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/v1/reports/%s", uuid.New().String()), nil)
	req.SetPathValue("id", uuid.New().String())
	w := httptest.NewRecorder()

	handler.HandleDelete(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
