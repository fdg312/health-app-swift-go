package notifications

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/fdg312/health-hub/internal/config"
	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/storage/memory"
)

func TestGenerateUsesUserSettingsThresholdsAndDailyLimit(t *testing.T) {
	ctx := context.Background()
	memStorage := memory.New()
	cfg := &config.Config{
		NotificationsMaxPerDay:     4,
		DefaultSleepMinMinutes:     420,
		DefaultStepsMin:            6000,
		DefaultActiveEnergyMinKcal: 250,
	}

	profiles, _ := memStorage.ListProfiles(ctx)
	ownerProfile := profiles[0]

	_, err := memStorage.UpsertSettings(ctx, ownerProfile.OwnerUserID, storage.Settings{
		NotificationsMaxPerDay: 1,
		MinSleepMinutes:        420,
		MinSteps:               20000,
		MinActiveEnergyKcal:    250,
		MorningCheckinMinute:   540,
		EveningCheckinMinute:   1260,
		VitaminsTimeMinute:     720,
	})
	if err != nil {
		t.Fatalf("upsert settings failed: %v", err)
	}

	service := NewService(
		memStorage.GetNotificationsStorage(),
		memStorage,
		memStorage.GetCheckinsStorage(),
		memStorage,
		memStorage,
		cfg,
	)

	now := time.Date(2026, 2, 13, 22, 0, 0, 0, time.UTC)
	date := now.Format("2006-01-02")

	payload, _ := json.Marshal(map[string]interface{}{
		"activity": map[string]interface{}{
			"steps": 10000,
		},
	})
	if err := memStorage.UpsertDailyMetric(ctx, ownerProfile.ID, date, payload); err != nil {
		t.Fatalf("upsert metric failed: %v", err)
	}

	resp, err := service.Generate(ctx, &GenerateRequest{
		ProfileID:      ownerProfile.ID,
		Date:           date,
		ClientTimeZone: "UTC",
		Now:            now,
		Thresholds: GenerateThresholds{
			SleepMinMinutes:     1,
			StepsMin:            1,
			ActiveEnergyMinKcal: 1,
		},
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if resp.Created != 1 {
		t.Fatalf("expected created=1 due to max_per_day override, got %d", resp.Created)
	}

	items, err := memStorage.GetNotificationsStorage().ListNotifications(ctx, ownerProfile.ID, false, 10, 0)
	if err != nil {
		t.Fatalf("list notifications failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected exactly 1 notification, got %d", len(items))
	}
	if items[0].Kind != "low_activity" {
		t.Fatalf("expected low_activity from settings steps threshold override, got %s", items[0].Kind)
	}
}

func TestGenerateQuietHoursSuppressesInfoReminders(t *testing.T) {
	ctx := context.Background()
	memStorage := memory.New()
	cfg := &config.Config{
		NotificationsMaxPerDay:     4,
		DefaultSleepMinMinutes:     420,
		DefaultStepsMin:            6000,
		DefaultActiveEnergyMinKcal: 250,
	}

	profiles, _ := memStorage.ListProfiles(ctx)
	ownerProfile := profiles[0]
	quietStart := 20 * 60
	quietEnd := 8 * 60
	tz := "UTC"

	_, err := memStorage.UpsertSettings(ctx, ownerProfile.OwnerUserID, storage.Settings{
		TimeZone:               &tz,
		QuietStartMinutes:      &quietStart,
		QuietEndMinutes:        &quietEnd,
		NotificationsMaxPerDay: 4,
		MinSleepMinutes:        420,
		MinSteps:               6000,
		MinActiveEnergyKcal:    250,
		MorningCheckinMinute:   540,
		EveningCheckinMinute:   1260,
		VitaminsTimeMinute:     720,
	})
	if err != nil {
		t.Fatalf("upsert settings failed: %v", err)
	}

	service := NewService(
		memStorage.GetNotificationsStorage(),
		memStorage,
		memStorage.GetCheckinsStorage(),
		memStorage,
		memStorage,
		cfg,
	)

	now := time.Date(2026, 2, 13, 22, 0, 0, 0, time.UTC)
	date := now.Format("2006-01-02")

	payload, _ := json.Marshal(map[string]interface{}{
		"sleep": map[string]interface{}{
			"total_minutes": 300,
		},
		"activity": map[string]interface{}{
			"steps": 10000,
		},
	})
	if err := memStorage.UpsertDailyMetric(ctx, ownerProfile.ID, date, payload); err != nil {
		t.Fatalf("upsert metric failed: %v", err)
	}

	resp, err := service.Generate(ctx, &GenerateRequest{
		ProfileID:      ownerProfile.ID,
		Date:           date,
		ClientTimeZone: "UTC",
		Now:            now,
		Thresholds: GenerateThresholds{
			SleepMinMinutes:     420,
			StepsMin:            6000,
			ActiveEnergyMinKcal: 250,
		},
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if resp.Created != 1 {
		t.Fatalf("expected created=1 (warn only), got %d", resp.Created)
	}

	items, err := memStorage.GetNotificationsStorage().ListNotifications(ctx, ownerProfile.ID, false, 10, 0)
	if err != nil {
		t.Fatalf("list notifications failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected exactly 1 notification, got %d", len(items))
	}
	if items[0].Kind != "low_sleep" || items[0].Severity != "warn" {
		t.Fatalf("expected only low_sleep warn, got kind=%s severity=%s", items[0].Kind, items[0].Severity)
	}
}
