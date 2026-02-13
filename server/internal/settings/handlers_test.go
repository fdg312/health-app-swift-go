package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fdg312/health-hub/internal/config"
	"github.com/fdg312/health-hub/internal/storage/memory"
	"github.com/fdg312/health-hub/internal/userctx"
)

func TestSettingsHandlersGetDefault(t *testing.T) {
	mem := memory.New()
	cfg := &config.Config{
		NotificationsMaxPerDay:     4,
		DefaultSleepMinMinutes:     420,
		DefaultStepsMin:            6000,
		DefaultActiveEnergyMinKcal: 250,
	}

	service := NewService(mem, cfg)
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/v1/settings", nil)
	req = req.WithContext(userctx.WithUserID(context.Background(), "user-a"))
	w := httptest.NewRecorder()
	handler.HandleGet(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp SettingsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if !resp.IsDefault {
		t.Fatalf("expected is_default=true")
	}
	if resp.Settings.MinSteps != 6000 {
		t.Fatalf("expected default min_steps=6000, got %d", resp.Settings.MinSteps)
	}
}

func TestSettingsHandlersPutAndGet(t *testing.T) {
	mem := memory.New()
	cfg := &config.Config{
		NotificationsMaxPerDay:     4,
		DefaultSleepMinMinutes:     420,
		DefaultStepsMin:            6000,
		DefaultActiveEnergyMinKcal: 250,
	}

	service := NewService(mem, cfg)
	handler := NewHandler(service)

	timeZone := "Europe/Moscow"
	quietStart := 1380
	quietEnd := 480
	payload := SettingsDTO{
		TimeZone:                  &timeZone,
		QuietStartMinutes:         &quietStart,
		QuietEndMinutes:           &quietEnd,
		NotificationsMaxPerDay:    2,
		MinSleepMinutes:           450,
		MinSteps:                  9000,
		MinActiveEnergyKcal:       350,
		MorningCheckinTimeMinutes: 500,
		EveningCheckinTimeMinutes: 1220,
		VitaminsTimeMinutes:       700,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/v1/settings", bytes.NewReader(body))
	req = req.WithContext(userctx.WithUserID(context.Background(), "user-b"))
	w := httptest.NewRecorder()
	handler.HandlePut(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	reqGet := httptest.NewRequest(http.MethodGet, "/v1/settings", nil)
	reqGet = reqGet.WithContext(userctx.WithUserID(context.Background(), "user-b"))
	wGet := httptest.NewRecorder()
	handler.HandleGet(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", wGet.Code)
	}

	var resp SettingsResponse
	if err := json.NewDecoder(wGet.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.IsDefault {
		t.Fatalf("expected is_default=false after put")
	}
	if resp.Settings.MinSteps != 9000 {
		t.Fatalf("expected min_steps=9000, got %d", resp.Settings.MinSteps)
	}
}

func TestSettingsHandlersUnauthorized(t *testing.T) {
	mem := memory.New()
	cfg := &config.Config{}
	service := NewService(mem, cfg)
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/v1/settings", nil)
	w := httptest.NewRecorder()
	handler.HandleGet(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestSettingsHandlersInvalidRequest(t *testing.T) {
	mem := memory.New()
	cfg := &config.Config{}
	service := NewService(mem, cfg)
	handler := NewHandler(service)

	invalidBody := []byte(`{"notifications_max_per_day": 999}`)
	req := httptest.NewRequest(http.MethodPut, "/v1/settings", bytes.NewReader(invalidBody))
	req = req.WithContext(userctx.WithUserID(context.Background(), "user-c"))
	w := httptest.NewRecorder()
	handler.HandlePut(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
}
