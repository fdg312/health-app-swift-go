package notifications

import (
	"context"
	"testing"
	"time"

	"github.com/fdg312/health-hub/internal/config"
	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/storage/memory"
)

func TestGenerateVitaminsReminder_WithScheduleAndNoIntake(t *testing.T) {
	ctx := context.Background()
	memStorage := memory.New()
	cfg := baseNotificationsConfig()

	profiles, _ := memStorage.ListProfiles(ctx)
	ownerProfile := profiles[0]

	supplement := &storage.Supplement{ProfileID: ownerProfile.ID, Name: "Магний"}
	if err := memStorage.CreateSupplement(ctx, supplement); err != nil {
		t.Fatalf("create supplement failed: %v", err)
	}
	if _, err := memStorage.UpsertSchedule(ctx, ownerProfile.OwnerUserID, ownerProfile.ID, storage.ScheduleUpsert{
		SupplementID: supplement.ID,
		TimeMinutes:  600,
		DaysMask:     127,
		IsEnabled:    true,
	}); err != nil {
		t.Fatalf("create schedule failed: %v", err)
	}

	if _, err := memStorage.UpsertSettings(ctx, ownerProfile.OwnerUserID, storage.Settings{
		NotificationsMaxPerDay: 4,
		MinSleepMinutes:        420,
		MinSteps:               6000,
		MinActiveEnergyKcal:    250,
		MorningCheckinMinute:   1439,
		EveningCheckinMinute:   1439,
		VitaminsTimeMinute:     600,
	}); err != nil {
		t.Fatalf("upsert settings failed: %v", err)
	}

	service := NewService(memStorage.GetNotificationsStorage(), memStorage, memStorage.GetCheckinsStorage(), memStorage, memStorage, cfg)

	now := time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC)
	date := now.Format("2006-01-02")

	resp, err := service.Generate(ctx, &GenerateRequest{
		ProfileID:      ownerProfile.ID,
		Date:           date,
		ClientTimeZone: "UTC",
		Now:            now,
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if resp.Created != 1 {
		t.Fatalf("expected created=1, got %d", resp.Created)
	}

	items, err := memStorage.GetNotificationsStorage().ListNotifications(ctx, ownerProfile.ID, false, 10, 0)
	if err != nil {
		t.Fatalf("list notifications failed: %v", err)
	}
	if len(items) != 1 || items[0].Kind != "vitamins_reminder" {
		t.Fatalf("expected vitamins_reminder, got %+v", items)
	}
}

func TestGenerateVitaminsReminder_NotGeneratedWhenTaken(t *testing.T) {
	ctx := context.Background()
	memStorage := memory.New()
	cfg := baseNotificationsConfig()

	profiles, _ := memStorage.ListProfiles(ctx)
	ownerProfile := profiles[0]

	supplement := &storage.Supplement{ProfileID: ownerProfile.ID, Name: "Витамин D"}
	if err := memStorage.CreateSupplement(ctx, supplement); err != nil {
		t.Fatalf("create supplement failed: %v", err)
	}
	if _, err := memStorage.UpsertSchedule(ctx, ownerProfile.OwnerUserID, ownerProfile.ID, storage.ScheduleUpsert{
		SupplementID: supplement.ID,
		TimeMinutes:  600,
		DaysMask:     127,
		IsEnabled:    true,
	}); err != nil {
		t.Fatalf("create schedule failed: %v", err)
	}

	now := time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC)
	date := now.Format("2006-01-02")
	if err := memStorage.UpsertSupplementIntake(ctx, &storage.SupplementIntake{
		ProfileID:    ownerProfile.ID,
		SupplementID: supplement.ID,
		TakenAt:      time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.UTC),
		Status:       "taken",
	}); err != nil {
		t.Fatalf("upsert intake failed: %v", err)
	}

	if _, err := memStorage.UpsertSettings(ctx, ownerProfile.OwnerUserID, storage.Settings{
		NotificationsMaxPerDay: 4,
		MinSleepMinutes:        420,
		MinSteps:               6000,
		MinActiveEnergyKcal:    250,
		MorningCheckinMinute:   1439,
		EveningCheckinMinute:   1439,
		VitaminsTimeMinute:     600,
	}); err != nil {
		t.Fatalf("upsert settings failed: %v", err)
	}

	service := NewService(memStorage.GetNotificationsStorage(), memStorage, memStorage.GetCheckinsStorage(), memStorage, memStorage, cfg)

	resp, err := service.Generate(ctx, &GenerateRequest{
		ProfileID:      ownerProfile.ID,
		Date:           date,
		ClientTimeZone: "UTC",
		Now:            now,
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if resp.Created != 0 {
		t.Fatalf("expected created=0, got %d", resp.Created)
	}
}

func TestGenerateVitaminsReminder_QuietHoursSuppresses(t *testing.T) {
	ctx := context.Background()
	memStorage := memory.New()
	cfg := baseNotificationsConfig()

	profiles, _ := memStorage.ListProfiles(ctx)
	ownerProfile := profiles[0]

	supplement := &storage.Supplement{ProfileID: ownerProfile.ID, Name: "Омега-3"}
	if err := memStorage.CreateSupplement(ctx, supplement); err != nil {
		t.Fatalf("create supplement failed: %v", err)
	}
	if _, err := memStorage.UpsertSchedule(ctx, ownerProfile.OwnerUserID, ownerProfile.ID, storage.ScheduleUpsert{
		SupplementID: supplement.ID,
		TimeMinutes:  600,
		DaysMask:     127,
		IsEnabled:    true,
	}); err != nil {
		t.Fatalf("create schedule failed: %v", err)
	}

	quietStart := 20 * 60
	quietEnd := 8 * 60
	tz := "UTC"
	if _, err := memStorage.UpsertSettings(ctx, ownerProfile.OwnerUserID, storage.Settings{
		TimeZone:               &tz,
		QuietStartMinutes:      &quietStart,
		QuietEndMinutes:        &quietEnd,
		NotificationsMaxPerDay: 4,
		MinSleepMinutes:        420,
		MinSteps:               6000,
		MinActiveEnergyKcal:    250,
		MorningCheckinMinute:   1439,
		EveningCheckinMinute:   1439,
		VitaminsTimeMinute:     600,
	}); err != nil {
		t.Fatalf("upsert settings failed: %v", err)
	}

	service := NewService(memStorage.GetNotificationsStorage(), memStorage, memStorage.GetCheckinsStorage(), memStorage, memStorage, cfg)
	now := time.Date(2026, 2, 13, 22, 0, 0, 0, time.UTC)
	date := now.Format("2006-01-02")

	resp, err := service.Generate(ctx, &GenerateRequest{ProfileID: ownerProfile.ID, Date: date, ClientTimeZone: "UTC", Now: now})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if resp.Created != 0 {
		t.Fatalf("expected created=0 in quiet hours, got %d", resp.Created)
	}
}

func TestGenerateVitaminsReminder_MaxPerDayZeroSuppresses(t *testing.T) {
	ctx := context.Background()
	memStorage := memory.New()
	cfg := baseNotificationsConfig()

	profiles, _ := memStorage.ListProfiles(ctx)
	ownerProfile := profiles[0]

	supplement := &storage.Supplement{ProfileID: ownerProfile.ID, Name: "Цинк"}
	if err := memStorage.CreateSupplement(ctx, supplement); err != nil {
		t.Fatalf("create supplement failed: %v", err)
	}
	if _, err := memStorage.UpsertSchedule(ctx, ownerProfile.OwnerUserID, ownerProfile.ID, storage.ScheduleUpsert{
		SupplementID: supplement.ID,
		TimeMinutes:  600,
		DaysMask:     127,
		IsEnabled:    true,
	}); err != nil {
		t.Fatalf("create schedule failed: %v", err)
	}
	if _, err := memStorage.UpsertSettings(ctx, ownerProfile.OwnerUserID, storage.Settings{
		NotificationsMaxPerDay: 0,
		MinSleepMinutes:        420,
		MinSteps:               6000,
		MinActiveEnergyKcal:    250,
		MorningCheckinMinute:   1439,
		EveningCheckinMinute:   1439,
		VitaminsTimeMinute:     600,
	}); err != nil {
		t.Fatalf("upsert settings failed: %v", err)
	}

	service := NewService(memStorage.GetNotificationsStorage(), memStorage, memStorage.GetCheckinsStorage(), memStorage, memStorage, cfg)
	now := time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC)
	date := now.Format("2006-01-02")

	resp, err := service.Generate(ctx, &GenerateRequest{ProfileID: ownerProfile.ID, Date: date, ClientTimeZone: "UTC", Now: now})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if resp.Created != 0 {
		t.Fatalf("expected created=0 when max_per_day=0, got %d", resp.Created)
	}
}

func baseNotificationsConfig() *config.Config {
	return &config.Config{
		NotificationsMaxPerDay:     4,
		DefaultSleepMinMinutes:     420,
		DefaultStepsMin:            6000,
		DefaultActiveEnergyMinKcal: 250,
	}
}
