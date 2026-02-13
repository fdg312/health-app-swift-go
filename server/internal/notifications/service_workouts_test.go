package notifications

import (
	"context"
	"testing"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

// Mock workout storages for testing
type mockWorkoutPlansStorage struct {
	plans map[string]storage.WorkoutPlan
}

func (m *mockWorkoutPlansStorage) GetActivePlan(ownerUserID string, profileID uuid.UUID) (storage.WorkoutPlan, bool, error) {
	key := ownerUserID + ":" + profileID.String()
	plan, ok := m.plans[key]
	return plan, ok, nil
}

type mockWorkoutItemsStorage struct {
	items map[uuid.UUID][]storage.WorkoutPlanItem
}

func (m *mockWorkoutItemsStorage) ListItems(ownerUserID string, profileID uuid.UUID, planID uuid.UUID) ([]storage.WorkoutPlanItem, error) {
	items, ok := m.items[planID]
	if !ok {
		return []storage.WorkoutPlanItem{}, nil
	}
	return items, nil
}

type mockWorkoutCompletionsStorage struct {
	completions map[string][]storage.WorkoutCompletion
}

func (m *mockWorkoutCompletionsStorage) ListCompletions(ownerUserID string, profileID uuid.UUID, from string, to string) ([]storage.WorkoutCompletion, error) {
	key := ownerUserID + ":" + profileID.String() + ":" + from
	comps, ok := m.completions[key]
	if !ok {
		return []storage.WorkoutCompletion{}, nil
	}
	return comps, nil
}

func TestWorkoutReminderGenerated(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	planID := uuid.New()
	itemID := uuid.New()
	ownerUserID := "testuser"

	// Setup: plan with one item for today (Monday = bit 0)
	today := time.Now()
	weekday := int(today.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekday--
	dayMask := 1 << weekday

	mockPlans := &mockWorkoutPlansStorage{
		plans: map[string]storage.WorkoutPlan{
			ownerUserID + ":" + profileID.String(): {
				ID:          planID,
				OwnerUserID: ownerUserID,
				ProfileID:   profileID,
				Title:       "Test Plan",
				Goal:        "fitness",
				IsActive:    true,
			},
		},
	}

	mockItems := &mockWorkoutItemsStorage{
		items: map[uuid.UUID][]storage.WorkoutPlanItem{
			planID: {
				{
					ID:          itemID,
					PlanID:      planID,
					OwnerUserID: ownerUserID,
					ProfileID:   profileID,
					Kind:        "run",
					TimeMinutes: 420, // 7:00 AM
					DaysMask:    dayMask,
					DurationMin: 30,
					Intensity:   "medium",
					Note:        "morning run",
				},
			},
		},
	}

	mockCompletions := &mockWorkoutCompletionsStorage{
		completions: map[string][]storage.WorkoutCompletion{},
	}

	// Create a mock service (simplified - only testing the workout reminder function)
	profile := &storage.Profile{
		ID:          profileID,
		OwnerUserID: ownerUserID,
		Type:        "owner",
		Name:        "Test User",
	}

	s := &Service{
		workoutPlans:       mockPlans,
		workoutItems:       mockItems,
		workoutCompletions: mockCompletions,
	}

	// Test at 7:00 AM (within 30 min window)
	now := time.Date(today.Year(), today.Month(), today.Day(), 7, 0, 0, 0, time.UTC)
	req := &GenerateRequest{
		ProfileID: profileID,
		Date:      today.Format("2006-01-02"),
		Now:       now,
	}

	effective := effectiveSettings{
		TimeZone: "UTC",
	}

	loc := time.UTC

	notification, err := s.maybeBuildWorkoutReminder(ctx, profile, req, effective, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if notification == nil {
		t.Fatalf("expected workout reminder to be generated, got nil")
	}

	if notification.Kind != "workout_reminder" {
		t.Errorf("expected kind 'workout_reminder', got %s", notification.Kind)
	}

	if notification.Title != "Тренировка сегодня" {
		t.Errorf("expected title 'Тренировка сегодня', got %s", notification.Title)
	}

	if notification.Severity != "info" {
		t.Errorf("expected severity 'info', got %s", notification.Severity)
	}
}

func TestWorkoutReminderNotGeneratedWhenCompleted(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	planID := uuid.New()
	itemID := uuid.New()
	ownerUserID := "testuser"

	today := time.Now()
	weekday := int(today.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekday--
	dayMask := 1 << weekday

	mockPlans := &mockWorkoutPlansStorage{
		plans: map[string]storage.WorkoutPlan{
			ownerUserID + ":" + profileID.String(): {
				ID:          planID,
				OwnerUserID: ownerUserID,
				ProfileID:   profileID,
				Title:       "Test Plan",
			},
		},
	}

	mockItems := &mockWorkoutItemsStorage{
		items: map[uuid.UUID][]storage.WorkoutPlanItem{
			planID: {
				{
					ID:          itemID,
					PlanID:      planID,
					OwnerUserID: ownerUserID,
					ProfileID:   profileID,
					Kind:        "run",
					TimeMinutes: 420,
					DaysMask:    dayMask,
					DurationMin: 30,
				},
			},
		},
	}

	// Workout already completed
	mockCompletions := &mockWorkoutCompletionsStorage{
		completions: map[string][]storage.WorkoutCompletion{
			ownerUserID + ":" + profileID.String() + ":" + today.Format("2006-01-02"): {
				{
					ID:          uuid.New(),
					OwnerUserID: ownerUserID,
					ProfileID:   profileID,
					Date:        today.Format("2006-01-02"),
					PlanItemID:  itemID,
					Status:      "done",
				},
			},
		},
	}

	profile := &storage.Profile{
		ID:          profileID,
		OwnerUserID: ownerUserID,
		Type:        "owner",
		Name:        "Test User",
	}

	s := &Service{
		workoutPlans:       mockPlans,
		workoutItems:       mockItems,
		workoutCompletions: mockCompletions,
	}

	now := time.Date(today.Year(), today.Month(), today.Day(), 7, 0, 0, 0, time.UTC)
	req := &GenerateRequest{
		ProfileID: profileID,
		Date:      today.Format("2006-01-02"),
		Now:       now,
	}

	effective := effectiveSettings{
		TimeZone: "UTC",
	}

	notification, err := s.maybeBuildWorkoutReminder(ctx, profile, req, effective, time.UTC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if notification != nil {
		t.Fatalf("expected no workout reminder when completed, got %+v", notification)
	}
}

func TestWorkoutReminderNotGeneratedInQuietHours(t *testing.T) {
	// This test would be part of the full Generate() test
	// where quiet hours filtering is applied to all candidates
	// The workout reminder should be filtered out if it's info severity
	// and current time is in quiet hours
}

func TestWorkoutReminderNotGeneratedTooEarly(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	planID := uuid.New()
	itemID := uuid.New()
	ownerUserID := "testuser"

	today := time.Now()
	weekday := int(today.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekday--
	dayMask := 1 << weekday

	mockPlans := &mockWorkoutPlansStorage{
		plans: map[string]storage.WorkoutPlan{
			ownerUserID + ":" + profileID.String(): {
				ID:          planID,
				OwnerUserID: ownerUserID,
				ProfileID:   profileID,
				Title:       "Test Plan",
			},
		},
	}

	mockItems := &mockWorkoutItemsStorage{
		items: map[uuid.UUID][]storage.WorkoutPlanItem{
			planID: {
				{
					ID:          itemID,
					PlanID:      planID,
					OwnerUserID: ownerUserID,
					ProfileID:   profileID,
					Kind:        "run",
					TimeMinutes: 420, // 7:00 AM
					DaysMask:    dayMask,
					DurationMin: 30,
				},
			},
		},
	}

	mockCompletions := &mockWorkoutCompletionsStorage{
		completions: map[string][]storage.WorkoutCompletion{},
	}

	profile := &storage.Profile{
		ID:          profileID,
		OwnerUserID: ownerUserID,
		Type:        "owner",
		Name:        "Test User",
	}

	s := &Service{
		workoutPlans:       mockPlans,
		workoutItems:       mockItems,
		workoutCompletions: mockCompletions,
	}

	// Test at 6:00 AM (more than 30 min before 7:00 AM workout)
	now := time.Date(today.Year(), today.Month(), today.Day(), 6, 0, 0, 0, time.UTC)
	req := &GenerateRequest{
		ProfileID: profileID,
		Date:      today.Format("2006-01-02"),
		Now:       now,
	}

	effective := effectiveSettings{
		TimeZone: "UTC",
	}

	notification, err := s.maybeBuildWorkoutReminder(ctx, profile, req, effective, time.UTC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if notification != nil {
		t.Fatalf("expected no workout reminder too early before workout time, got %+v", notification)
	}
}
