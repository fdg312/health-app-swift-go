package workouts

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/storage/memory"
	"github.com/fdg312/health-hub/internal/userctx"
	"github.com/google/uuid"
)

func TestWorkoutsReplacePlanAndGet(t *testing.T) {
	ctx := context.Background()
	mem := memory.New()

	profileA := uuid.New()
	if err := mem.CreateProfile(ctx, &storage.Profile{ID: profileA, OwnerUserID: "userA", Type: "owner", Name: "User A"}); err != nil {
		t.Fatalf("create profileA: %v", err)
	}

	h := NewHandlers(NewService(
		mem.GetWorkoutPlansStorage(),
		mem.GetWorkoutPlanItemsStorage(),
		mem.GetWorkoutCompletionsStorage(),
		mem,
		nil, // no feed service for this test
	))

	// Replace plan
	replaceBody, _ := json.Marshal(ReplaceItemsRequest{
		ProfileID: profileA,
		Title:     "Endurance Plan",
		Goal:      "endurance",
		Replace:   true,
		Items: []ItemUpsertRequest{
			{
				Kind:        "run",
				TimeMinutes: 420,
				DaysMask:    62, // Mon-Fri (bits 0-4)
				DurationMin: 30,
				Intensity:   "medium",
				Note:        "easy pace",
				Details:     json.RawMessage(`{}`),
			},
			{
				Kind:        "strength",
				TimeMinutes: 1140,
				DaysMask:    20, // Tue, Thu (bits 1,3)
				DurationMin: 40,
				Intensity:   "high",
				Note:        "upper body",
				Details:     json.RawMessage(`{}`),
			},
		},
	})

	replaceReq := httptest.NewRequest(http.MethodPut, "/v1/workouts/plan/replace", bytes.NewReader(replaceBody))
	replaceReq = replaceReq.WithContext(userctx.WithUserID(context.Background(), "userA"))
	replaceW := httptest.NewRecorder()
	h.HandleReplacePlan(replaceW, replaceReq)

	if replaceW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", replaceW.Code, replaceW.Body.String())
	}

	var replaceResp ReplaceItemsResponse
	if err := json.NewDecoder(replaceW.Body).Decode(&replaceResp); err != nil {
		t.Fatalf("decode replace response: %v", err)
	}
	if replaceResp.Plan.Title != "Endurance Plan" {
		t.Fatalf("expected title 'Endurance Plan', got %s", replaceResp.Plan.Title)
	}
	if len(replaceResp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(replaceResp.Items))
	}

	// Get plan
	getReq := httptest.NewRequest(http.MethodGet, "/v1/workouts/plan?profile_id="+profileA.String(), nil)
	getReq = getReq.WithContext(userctx.WithUserID(context.Background(), "userA"))
	getW := httptest.NewRecorder()
	h.HandleGetPlan(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", getW.Code, getW.Body.String())
	}

	var getResp GetPlanResponse
	if err := json.NewDecoder(getW.Body).Decode(&getResp); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if getResp.Plan == nil {
		t.Fatalf("expected plan, got nil")
	}
	if len(getResp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(getResp.Items))
	}
}

func TestWorkoutsTodayWithPlannedItems(t *testing.T) {
	ctx := context.Background()
	mem := memory.New()

	profileA := uuid.New()
	if err := mem.CreateProfile(ctx, &storage.Profile{ID: profileA, OwnerUserID: "userA", Type: "owner", Name: "User A"}); err != nil {
		t.Fatalf("create profileA: %v", err)
	}

	h := NewHandlers(NewService(
		mem.GetWorkoutPlansStorage(),
		mem.GetWorkoutPlanItemsStorage(),
		mem.GetWorkoutCompletionsStorage(),
		mem,
		nil,
	))

	// Create plan with items for Monday (bit 0)
	today := time.Now()
	weekday := int(today.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday
	}
	weekday-- // 0-indexed

	dayMask := 1 << weekday

	replaceBody, _ := json.Marshal(ReplaceItemsRequest{
		ProfileID: profileA,
		Title:     "Daily Plan",
		Goal:      "fitness",
		Replace:   true,
		Items: []ItemUpsertRequest{
			{
				Kind:        "run",
				TimeMinutes: 420,
				DaysMask:    dayMask,
				DurationMin: 30,
				Intensity:   "medium",
				Note:        "morning run",
				Details:     json.RawMessage(`{}`),
			},
		},
	})

	replaceReq := httptest.NewRequest(http.MethodPut, "/v1/workouts/plan/replace", bytes.NewReader(replaceBody))
	replaceReq = replaceReq.WithContext(userctx.WithUserID(context.Background(), "userA"))
	replaceW := httptest.NewRecorder()
	h.HandleReplacePlan(replaceW, replaceReq)

	if replaceW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", replaceW.Code)
	}

	// Get today
	date := today.Format("2006-01-02")
	todayReq := httptest.NewRequest(http.MethodGet, "/v1/workouts/today?profile_id="+profileA.String()+"&date="+date, nil)
	todayReq = todayReq.WithContext(userctx.WithUserID(context.Background(), "userA"))
	todayW := httptest.NewRecorder()
	h.HandleGetToday(todayW, todayReq)

	if todayW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", todayW.Code, todayW.Body.String())
	}

	var todayResp TodayResponse
	if err := json.NewDecoder(todayW.Body).Decode(&todayResp); err != nil {
		t.Fatalf("decode today response: %v", err)
	}

	if len(todayResp.Planned) != 1 {
		t.Fatalf("expected 1 planned item for today, got %d", len(todayResp.Planned))
	}
	if todayResp.Planned[0].Kind != "run" {
		t.Fatalf("expected kind 'run', got %s", todayResp.Planned[0].Kind)
	}
}

func TestWorkoutsCompletionAndTodayStatus(t *testing.T) {
	ctx := context.Background()
	mem := memory.New()

	profileA := uuid.New()
	if err := mem.CreateProfile(ctx, &storage.Profile{ID: profileA, OwnerUserID: "userA", Type: "owner", Name: "User A"}); err != nil {
		t.Fatalf("create profileA: %v", err)
	}

	h := NewHandlers(NewService(
		mem.GetWorkoutPlansStorage(),
		mem.GetWorkoutPlanItemsStorage(),
		mem.GetWorkoutCompletionsStorage(),
		mem,
		nil,
	))

	// Create plan
	replaceBody, _ := json.Marshal(ReplaceItemsRequest{
		ProfileID: profileA,
		Title:     "Test Plan",
		Goal:      "test",
		Replace:   true,
		Items: []ItemUpsertRequest{
			{
				Kind:        "run",
				TimeMinutes: 420,
				DaysMask:    127, // all days
				DurationMin: 30,
				Intensity:   "medium",
				Note:        "",
				Details:     json.RawMessage(`{}`),
			},
		},
	})

	replaceReq := httptest.NewRequest(http.MethodPut, "/v1/workouts/plan/replace", bytes.NewReader(replaceBody))
	replaceReq = replaceReq.WithContext(userctx.WithUserID(context.Background(), "userA"))
	replaceW := httptest.NewRecorder()
	h.HandleReplacePlan(replaceW, replaceReq)

	if replaceW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", replaceW.Code)
	}

	var replaceResp ReplaceItemsResponse
	json.NewDecoder(replaceW.Body).Decode(&replaceResp)
	planItemID := replaceResp.Items[0].ID

	// Check today before completion
	date := time.Now().Format("2006-01-02")
	todayReq1 := httptest.NewRequest(http.MethodGet, "/v1/workouts/today?profile_id="+profileA.String()+"&date="+date, nil)
	todayReq1 = todayReq1.WithContext(userctx.WithUserID(context.Background(), "userA"))
	todayW1 := httptest.NewRecorder()
	h.HandleGetToday(todayW1, todayReq1)

	var todayResp1 TodayResponse
	json.NewDecoder(todayW1.Body).Decode(&todayResp1)
	if todayResp1.IsDone {
		t.Fatalf("expected isDone=false before completion")
	}

	// Upsert completion
	completionBody, _ := json.Marshal(UpsertCompletionRequest{
		ProfileID:  profileA,
		Date:       date,
		PlanItemID: planItemID,
		Status:     "done",
		Note:       "completed",
	})

	completionReq := httptest.NewRequest(http.MethodPost, "/v1/workouts/completions", bytes.NewReader(completionBody))
	completionReq = completionReq.WithContext(userctx.WithUserID(context.Background(), "userA"))
	completionW := httptest.NewRecorder()
	h.HandleUpsertCompletion(completionW, completionReq)

	if completionW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", completionW.Code, completionW.Body.String())
	}

	// Check today after completion
	todayReq2 := httptest.NewRequest(http.MethodGet, "/v1/workouts/today?profile_id="+profileA.String()+"&date="+date, nil)
	todayReq2 = todayReq2.WithContext(userctx.WithUserID(context.Background(), "userA"))
	todayW2 := httptest.NewRecorder()
	h.HandleGetToday(todayW2, todayReq2)

	var todayResp2 TodayResponse
	json.NewDecoder(todayW2.Body).Decode(&todayResp2)
	if !todayResp2.IsDone {
		t.Fatalf("expected isDone=true after completion")
	}
	if len(todayResp2.Completions) != 1 {
		t.Fatalf("expected 1 completion, got %d", len(todayResp2.Completions))
	}
}

func TestWorkoutsOwnershipIsolation(t *testing.T) {
	ctx := context.Background()
	mem := memory.New()

	profileA := uuid.New()
	profileB := uuid.New()
	if err := mem.CreateProfile(ctx, &storage.Profile{ID: profileA, OwnerUserID: "userA", Type: "owner", Name: "User A"}); err != nil {
		t.Fatalf("create profileA: %v", err)
	}
	if err := mem.CreateProfile(ctx, &storage.Profile{ID: profileB, OwnerUserID: "userB", Type: "owner", Name: "User B"}); err != nil {
		t.Fatalf("create profileB: %v", err)
	}

	h := NewHandlers(NewService(
		mem.GetWorkoutPlansStorage(),
		mem.GetWorkoutPlanItemsStorage(),
		mem.GetWorkoutCompletionsStorage(),
		mem,
		nil,
	))

	// UserA creates plan
	replaceBody, _ := json.Marshal(ReplaceItemsRequest{
		ProfileID: profileA,
		Title:     "UserA Plan",
		Goal:      "fitness",
		Replace:   true,
		Items: []ItemUpsertRequest{
			{
				Kind:        "run",
				TimeMinutes: 420,
				DaysMask:    127,
				DurationMin: 30,
				Intensity:   "medium",
				Note:        "",
				Details:     json.RawMessage(`{}`),
			},
		},
	})

	replaceReq := httptest.NewRequest(http.MethodPut, "/v1/workouts/plan/replace", bytes.NewReader(replaceBody))
	replaceReq = replaceReq.WithContext(userctx.WithUserID(context.Background(), "userA"))
	replaceW := httptest.NewRecorder()
	h.HandleReplacePlan(replaceW, replaceReq)

	if replaceW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", replaceW.Code)
	}

	// UserB tries to access UserA's plan - should get 404
	getReq := httptest.NewRequest(http.MethodGet, "/v1/workouts/plan?profile_id="+profileA.String(), nil)
	getReq = getReq.WithContext(userctx.WithUserID(context.Background(), "userB"))
	getW := httptest.NewRecorder()
	h.HandleGetPlan(getW, getReq)

	if getW.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-user access, got %d", getW.Code)
	}
}

func TestWorkoutsValidationTooManyItemsPerDay(t *testing.T) {
	ctx := context.Background()
	mem := memory.New()

	profileA := uuid.New()
	if err := mem.CreateProfile(ctx, &storage.Profile{ID: profileA, OwnerUserID: "userA", Type: "owner", Name: "User A"}); err != nil {
		t.Fatalf("create profileA: %v", err)
	}

	h := NewHandlers(NewService(
		mem.GetWorkoutPlansStorage(),
		mem.GetWorkoutPlanItemsStorage(),
		mem.GetWorkoutCompletionsStorage(),
		mem,
		nil,
	))

	// Try to create 5 items on Monday (bit 0) - should fail (max 4 per day)
	items := []ItemUpsertRequest{}
	for i := 0; i < 5; i++ {
		items = append(items, ItemUpsertRequest{
			Kind:        "run",
			TimeMinutes: 420 + i*60,
			DaysMask:    1, // Monday only
			DurationMin: 30,
			Intensity:   "medium",
			Note:        "",
			Details:     json.RawMessage(`{}`),
		})
	}

	replaceBody, _ := json.Marshal(ReplaceItemsRequest{
		ProfileID: profileA,
		Title:     "Overloaded Plan",
		Goal:      "test",
		Replace:   true,
		Items:     items,
	})

	replaceReq := httptest.NewRequest(http.MethodPut, "/v1/workouts/plan/replace", bytes.NewReader(replaceBody))
	replaceReq = replaceReq.WithContext(userctx.WithUserID(context.Background(), "userA"))
	replaceW := httptest.NewRecorder()
	h.HandleReplacePlan(replaceW, replaceReq)

	if replaceW.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for too many items per day, got %d body=%s", replaceW.Code, replaceW.Body.String())
	}
}

func TestWorkoutsValidationReplaceNotTrue(t *testing.T) {
	ctx := context.Background()
	mem := memory.New()

	profileA := uuid.New()
	if err := mem.CreateProfile(ctx, &storage.Profile{ID: profileA, OwnerUserID: "userA", Type: "owner", Name: "User A"}); err != nil {
		t.Fatalf("create profileA: %v", err)
	}

	h := NewHandlers(NewService(
		mem.GetWorkoutPlansStorage(),
		mem.GetWorkoutPlanItemsStorage(),
		mem.GetWorkoutCompletionsStorage(),
		mem,
		nil,
	))

	replaceBody, _ := json.Marshal(ReplaceItemsRequest{
		ProfileID: profileA,
		Title:     "Test Plan",
		Goal:      "test",
		Replace:   false, // should fail
		Items: []ItemUpsertRequest{
			{
				Kind:        "run",
				TimeMinutes: 420,
				DaysMask:    127,
				DurationMin: 30,
				Intensity:   "medium",
				Note:        "",
				Details:     json.RawMessage(`{}`),
			},
		},
	})

	replaceReq := httptest.NewRequest(http.MethodPut, "/v1/workouts/plan/replace", bytes.NewReader(replaceBody))
	replaceReq = replaceReq.WithContext(userctx.WithUserID(context.Background(), "userA"))
	replaceW := httptest.NewRecorder()
	h.HandleReplacePlan(replaceW, replaceReq)

	if replaceW.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for replace=false, got %d", replaceW.Code)
	}
}
