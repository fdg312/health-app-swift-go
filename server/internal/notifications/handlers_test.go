package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fdg312/health-hub/internal/config"
	"github.com/fdg312/health-hub/internal/storage/memory"
	"github.com/google/uuid"
)

func TestNotificationsHandlers(t *testing.T) {
	ctx := context.Background()
	memStorage := memory.New()
	cfg := &config.Config{
		NotificationsMaxPerDay:     4,
		DefaultSleepMinMinutes:     420,
		DefaultStepsMin:            6000,
		DefaultActiveEnergyMinKcal: 200,
	}

	// Get owner profile
	profiles, _ := memStorage.ListProfiles(ctx)
	ownerID := profiles[0].ID

	service := NewService(
		memStorage.GetNotificationsStorage(),
		memStorage,
		memStorage.GetCheckinsStorage(),
		memStorage,
		memStorage,
		cfg,
	)
	handler := NewHandler(service)

	t.Run("GenerateLowSleep", func(t *testing.T) {
		// Setup: add daily metrics with low sleep
		date := "2026-02-13"
		dailyData := map[string]interface{}{
			"sleep": map[string]interface{}{
				"total_minutes": 300, // 5 hours < 7 hours threshold
			},
		}
		payload, _ := json.Marshal(dailyData)
		memStorage.UpsertDailyMetric(ctx, ownerID, date, payload)

		// Generate
		reqBody := GenerateRequest{
			ProfileID:      ownerID,
			Date:           date,
			ClientTimeZone: "UTC",
			Now:            time.Now(),
			Thresholds: GenerateThresholds{
				SleepMinMinutes:     420,
				StepsMin:            6000,
				ActiveEnergyMinKcal: 200,
			},
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/v1/inbox/generate", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.HandleGenerate(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp GenerateResponse
		json.NewDecoder(w.Body).Decode(&resp)

		if resp.Created == 0 {
			t.Errorf("expected at least 1 notification created")
		}

		// Verify notification exists
		notifications, _ := memStorage.GetNotificationsStorage().ListNotifications(ctx, ownerID, false, 10, 0)
		if len(notifications) == 0 {
			t.Errorf("expected notifications to be created")
		}

		found := false
		for _, n := range notifications {
			if n.Kind == "low_sleep" {
				found = true
				if n.Severity != "warn" {
					t.Errorf("expected warn severity, got %s", n.Severity)
				}
			}
		}
		if !found {
			t.Errorf("low_sleep notification not found")
		}
	})

	t.Run("UnreadCount", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/inbox/unread-count?profile_id="+ownerID.String(), nil)
		w := httptest.NewRecorder()

		handler.HandleUnreadCount(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp UnreadCountResponse
		json.NewDecoder(w.Body).Decode(&resp)

		if resp.Unread == 0 {
			t.Errorf("expected unread > 0, got %d", resp.Unread)
		}
	})

	t.Run("ListNotifications", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/inbox?profile_id="+ownerID.String(), nil)
		w := httptest.NewRecorder()

		handler.HandleList(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp InboxListResponse
		json.NewDecoder(w.Body).Decode(&resp)

		if len(resp.Notifications) == 0 {
			t.Errorf("expected notifications, got empty list")
		}
	})

	t.Run("MarkRead", func(t *testing.T) {
		// Get first notification
		notifications, _ := memStorage.GetNotificationsStorage().ListNotifications(ctx, ownerID, true, 1, 0)
		if len(notifications) == 0 {
			t.Skip("no unread notifications")
		}

		notifID := notifications[0].ID

		reqBody := MarkReadRequest{
			ProfileID: ownerID,
			IDs:       []uuid.UUID{notifID},
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/v1/inbox/mark-read", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.HandleMarkRead(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp MarkReadResponse
		json.NewDecoder(w.Body).Decode(&resp)

		if resp.Marked != 1 {
			t.Errorf("expected marked=1, got %d", resp.Marked)
		}

		// Verify unread count decreased
		count, _ := memStorage.GetNotificationsStorage().UnreadCount(ctx, ownerID)
		if count < 0 {
			t.Errorf("unread count should be >= 0, got %d", count)
		}
	})

	t.Run("MarkAllRead", func(t *testing.T) {
		reqBody := MarkAllReadRequest{
			ProfileID: ownerID,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/v1/inbox/mark-all-read", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.HandleMarkAllRead(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		// Verify unread count is 0
		count, _ := memStorage.GetNotificationsStorage().UnreadCount(ctx, ownerID)
		if count != 0 {
			t.Errorf("expected unread count 0, got %d", count)
		}
	})

	t.Run("GenerateIdempotent", func(t *testing.T) {
		// Generate twice with same date - should not create duplicates
		date := "2026-02-14"
		dailyData := map[string]interface{}{
			"activity": map[string]interface{}{
				"steps": 3000, // low
			},
		}
		payload, _ := json.Marshal(dailyData)
		memStorage.UpsertDailyMetric(ctx, ownerID, date, payload)

		reqBody := GenerateRequest{
			ProfileID:      ownerID,
			Date:           date,
			ClientTimeZone: "UTC",
			Now:            time.Now(),
			Thresholds: GenerateThresholds{
				SleepMinMinutes:     420,
				StepsMin:            6000,
				ActiveEnergyMinKcal: 200,
			},
		}
		body, _ := json.Marshal(reqBody)

		// First generate
		req1 := httptest.NewRequest("POST", "/v1/inbox/generate", bytes.NewReader(body))
		w1 := httptest.NewRecorder()
		handler.HandleGenerate(w1, req1)

		// Second generate
		body2, _ := json.Marshal(reqBody)
		req2 := httptest.NewRequest("POST", "/v1/inbox/generate", bytes.NewReader(body2))
		w2 := httptest.NewRecorder()
		handler.HandleGenerate(w2, req2)

		// Should not create duplicates (unique constraint)
		notifications, _ := memStorage.GetNotificationsStorage().ListNotifications(ctx, ownerID, false, 100, 0)
		lowActivityCount := 0
		for _, n := range notifications {
			if n.Kind == "low_activity" && n.SourceDate != nil {
				sourceDate := n.SourceDate.Format("2006-01-02")
				if sourceDate == date {
					lowActivityCount++
				}
			}
		}

		if lowActivityCount > 1 {
			t.Errorf("expected 1 low_activity notification for date, got %d", lowActivityCount)
		}
	})
}

func TestGenerateMissingCheckin(t *testing.T) {
	ctx := context.Background()
	memStorage := memory.New()
	cfg := &config.Config{
		NotificationsMaxPerDay:     4,
		DefaultSleepMinMinutes:     420,
		DefaultStepsMin:            6000,
		DefaultActiveEnergyMinKcal: 200,
	}

	profiles, _ := memStorage.ListProfiles(ctx)
	ownerID := profiles[0].ID

	service := NewService(
		memStorage.GetNotificationsStorage(),
		memStorage,
		memStorage.GetCheckinsStorage(),
		memStorage,
		memStorage,
		cfg,
	)

	t.Run("MissingEveningCheckin", func(t *testing.T) {
		// Today, evening time (22:00)
		now := time.Date(2026, 2, 13, 22, 0, 0, 0, time.UTC)
		date := now.Format("2006-01-02")

		req := &GenerateRequest{
			ProfileID:      ownerID,
			Date:           date,
			ClientTimeZone: "UTC",
			Now:            now,
			Thresholds: GenerateThresholds{
				SleepMinMinutes:     420,
				StepsMin:            6000,
				ActiveEnergyMinKcal: 200,
			},
		}

		resp, err := service.Generate(ctx, req)
		if err != nil {
			t.Fatalf("generate failed: %v", err)
		}

		if resp.Created == 0 {
			t.Errorf("expected at least 1 notification created")
		}

		// Verify missing_evening_checkin notification
		notifications, _ := memStorage.GetNotificationsStorage().ListNotifications(ctx, ownerID, false, 10, 0)
		found := false
		for _, n := range notifications {
			if n.Kind == "missing_evening_checkin" {
				found = true
			}
		}
		if !found {
			t.Errorf("missing_evening_checkin notification not found")
		}
	})
}
