package mealplans

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
)

type mockMealPlansRepo struct {
	plans []storage.MealPlan
	items []storage.MealPlanItem
}

func (m *mockMealPlansRepo) GetActive(ctx context.Context, ownerUserID, profileID string) (storage.MealPlan, []storage.MealPlanItem, bool, error) {
	for _, p := range m.plans {
		if p.OwnerUserID == ownerUserID && p.ProfileID == profileID && p.IsActive {
			var planItems []storage.MealPlanItem
			for _, item := range m.items {
				if item.PlanID == p.ID {
					planItems = append(planItems, item)
				}
			}
			return p, planItems, true, nil
		}
	}
	return storage.MealPlan{}, nil, false, nil
}

func (m *mockMealPlansRepo) ReplaceActive(ctx context.Context, ownerUserID, profileID, title string, items []storage.MealPlanItemUpsert) (storage.MealPlan, []storage.MealPlanItem, error) {
	// Delete existing active plan
	var oldPlanID string
	for i, p := range m.plans {
		if p.OwnerUserID == ownerUserID && p.ProfileID == profileID && p.IsActive {
			oldPlanID = p.ID
			m.plans = append(m.plans[:i], m.plans[i+1:]...)
			break
		}
	}

	// Remove old items
	if oldPlanID != "" {
		newItems := []storage.MealPlanItem{}
		for _, item := range m.items {
			if item.PlanID != oldPlanID {
				newItems = append(newItems, item)
			}
		}
		m.items = newItems
	}

	// Create new plan
	plan := storage.MealPlan{
		ID:          fmt.Sprintf("plan%d", len(m.plans)+1),
		OwnerUserID: ownerUserID,
		ProfileID:   profileID,
		Title:       title,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.plans = append(m.plans, plan)

	// Create items
	var createdItems []storage.MealPlanItem
	for _, upsert := range items {
		item := storage.MealPlanItem{
			ID:             fmt.Sprintf("item%d", len(m.items)+1),
			OwnerUserID:    ownerUserID,
			ProfileID:      profileID,
			PlanID:         plan.ID,
			DayIndex:       upsert.DayIndex,
			MealSlot:       upsert.MealSlot,
			Title:          upsert.Title,
			Notes:          upsert.Notes,
			ApproxKcal:     upsert.ApproxKcal,
			ApproxProteinG: upsert.ApproxProteinG,
			ApproxFatG:     upsert.ApproxFatG,
			ApproxCarbsG:   upsert.ApproxCarbsG,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		m.items = append(m.items, item)
		createdItems = append(createdItems, item)
	}

	return plan, createdItems, nil
}

func (m *mockMealPlansRepo) GetToday(ctx context.Context, ownerUserID, profileID string, date time.Time) ([]storage.MealPlanItem, error) {
	// Calculate day index (0 = Sunday, 1 = Monday, etc.)
	dayIndex := int(date.Weekday())

	var result []storage.MealPlanItem
	for _, p := range m.plans {
		if p.OwnerUserID == ownerUserID && p.ProfileID == profileID && p.IsActive {
			for _, item := range m.items {
				if item.PlanID == p.ID && item.DayIndex == dayIndex {
					result = append(result, item)
				}
			}
			break
		}
	}
	return result, nil
}

func (m *mockMealPlansRepo) DeleteActive(ctx context.Context, ownerUserID, profileID string) error {
	var planID string
	for i, p := range m.plans {
		if p.OwnerUserID == ownerUserID && p.ProfileID == profileID && p.IsActive {
			planID = p.ID
			m.plans = append(m.plans[:i], m.plans[i+1:]...)
			break
		}
	}

	if planID != "" {
		newItems := []storage.MealPlanItem{}
		for _, item := range m.items {
			if item.PlanID != planID {
				newItems = append(newItems, item)
			}
		}
		m.items = newItems
	}

	return nil
}

func TestHandleReplace_Success(t *testing.T) {
	repo := &mockMealPlansRepo{}
	service := NewService(repo)
	handler := NewHandler(service)

	reqBody := ReplaceMealPlanRequest{
		ProfileID: "profile1",
		Title:     "My Meal Plan",
		Items: []MealPlanItemUpsertInput{
			{
				DayIndex:       0,
				MealSlot:       "breakfast",
				Title:          "Oatmeal",
				Notes:          "",
				ApproxKcal:     300,
				ApproxProteinG: 10,
				ApproxFatG:     5,
				ApproxCarbsG:   50,
			},
			{
				DayIndex:       0,
				MealSlot:       "lunch",
				Title:          "Chicken Salad",
				Notes:          "",
				ApproxKcal:     450,
				ApproxProteinG: 35,
				ApproxFatG:     20,
				ApproxCarbsG:   30,
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/v1/meal/plan/replace", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), "user_id", "user1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleReplace(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response GetMealPlanResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Plan == nil {
		t.Fatal("expected plan to be returned")
	}

	if response.Plan.Title != "My Meal Plan" {
		t.Errorf("expected title 'My Meal Plan', got '%s'", response.Plan.Title)
	}

	if len(response.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(response.Items))
	}
}

func TestHandleReplace_DuplicateDaySlot(t *testing.T) {
	repo := &mockMealPlansRepo{}
	service := NewService(repo)
	handler := NewHandler(service)

	reqBody := ReplaceMealPlanRequest{
		ProfileID: "profile1",
		Title:     "Duplicate Test",
		Items: []MealPlanItemUpsertInput{
			{
				DayIndex:   0,
				MealSlot:   "breakfast",
				Title:      "Meal 1",
				ApproxKcal: 300,
			},
			{
				DayIndex:   0,
				MealSlot:   "breakfast", // Duplicate!
				Title:      "Meal 2",
				ApproxKcal: 400,
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/v1/meal/plan/replace", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), "user_id", "user1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleReplace(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for duplicate day+slot, got %d. Response body: %s", w.Code, w.Body.String())
	}
}

func TestHandleReplace_MaxItems(t *testing.T) {
	repo := &mockMealPlansRepo{}
	service := NewService(repo)
	handler := NewHandler(service)

	// Create 29 items (max is 28)
	items := []MealPlanItemUpsertInput{}
	for day := 0; day < 7; day++ {
		for slot := 0; slot < 4; slot++ {
			slotName := []string{"breakfast", "lunch", "dinner", "snack"}[slot]
			items = append(items, MealPlanItemUpsertInput{
				DayIndex:   day,
				MealSlot:   slotName,
				Title:      "Meal",
				ApproxKcal: 300,
			})
		}
	}
	// Add one extra to exceed limit
	items = append(items, MealPlanItemUpsertInput{
		DayIndex:   0,
		MealSlot:   "extra", // This would be the 29th item
		Title:      "Extra Meal",
		ApproxKcal: 100,
	})

	reqBody := ReplaceMealPlanRequest{
		ProfileID: "profile1",
		Title:     "Too Many Items",
		Items:     items,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/v1/meal/plan/replace", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), "user_id", "user1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleReplace(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for exceeding max items, got %d", w.Code)
	}
}

func TestHandleGetToday_CorrectDayIndex(t *testing.T) {
	repo := &mockMealPlansRepo{}
	service := NewService(repo)
	handler := NewHandler(service)

	// Create a plan with items for different days
	plan := storage.MealPlan{
		ID:          "plan1",
		OwnerUserID: "user1",
		ProfileID:   "profile1",
		Title:       "Weekly Plan",
		IsActive:    true,
		FromDate:    nil, // Starts from "today" (Sunday)
	}
	repo.plans = []storage.MealPlan{plan}

	// Add items for day 0 (Sunday) and day 1 (Monday)
	repo.items = []storage.MealPlanItem{
		{
			ID:        "item1",
			PlanID:    "plan1",
			ProfileID: "profile1",
			DayIndex:  0,
			MealSlot:  "breakfast",
			Title:     "Sunday Breakfast",
		},
		{
			ID:        "item2",
			PlanID:    "plan1",
			ProfileID: "profile1",
			DayIndex:  1,
			MealSlot:  "breakfast",
			Title:     "Monday Breakfast",
		},
	}

	// Query for a specific Sunday date (2024-01-07 is a Sunday)
	req := httptest.NewRequest(http.MethodGet, "/v1/meal/today?profile_id=profile1&date=2024-01-07", nil)
	ctx := context.WithValue(req.Context(), "user_id", "user1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleGetToday(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response GetTodayResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should return day 0 items (Sunday)
	if len(response.Items) != 1 {
		t.Errorf("expected 1 item for Sunday (day 0), got %d", len(response.Items))
	}

	if len(response.Items) > 0 && response.Items[0].Title != "Sunday Breakfast" {
		t.Errorf("expected 'Sunday Breakfast', got '%s'", response.Items[0].Title)
	}
}

func TestHandleDelete_Ownership(t *testing.T) {
	repo := &mockMealPlansRepo{}
	service := NewService(repo)
	handler := NewHandler(service)

	// Create a plan for user1
	plan := storage.MealPlan{
		ID:          "plan1",
		OwnerUserID: "user1",
		ProfileID:   "profile1",
		Title:       "User1's Plan",
		IsActive:    true,
	}
	repo.plans = []storage.MealPlan{plan}

	// Try to delete as user2 (different owner)
	req := httptest.NewRequest(http.MethodDelete, "/v1/meal/plan?profile_id=profile1", nil)
	ctx := context.WithValue(req.Context(), "user_id", "user2")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleDelete(w, req)

	// Should succeed (returns 204) but not actually delete since ownership doesn't match
	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}

	// Verify plan still exists (ownership protection in service/repo layer)
	if len(repo.plans) != 1 {
		t.Errorf("plan should still exist due to ownership protection")
	}
}

func TestHandleGet_ReturnsEmptyWhenNoPlan(t *testing.T) {
	repo := &mockMealPlansRepo{}
	service := NewService(repo)
	handler := NewHandler(service)

	// No plans in repo

	req := httptest.NewRequest(http.MethodGet, "/v1/meal/plan?profile_id=profile1", nil)
	ctx := context.WithValue(req.Context(), "user_id", "user1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleGet(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response GetMealPlanResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Plan != nil {
		t.Error("expected plan to be nil when no active plan exists")
	}

	if len(response.Items) != 0 {
		t.Errorf("expected 0 items when no plan exists, got %d", len(response.Items))
	}
}
