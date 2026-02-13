package intakes

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

func setupTestService() (*Service, uuid.UUID) {
	memStorage := memory.New()
	cfg := &config.Config{
		IntakesMaxWaterMlPerDay:  8000,
		IntakesWaterDefaultAddMl: 250,
		IntakesMaxSupplements:    100,
	}

	ctx := context.Background()
	profiles, _ := memStorage.ListProfiles(ctx)
	ownerID := profiles[0].ID

	service := NewService(
		memStorage.GetSupplementsStorage(),
		memStorage.GetIntakesStorage(),
		memStorage,
		cfg,
	)

	return service, ownerID
}

func TestSupplementsHandlers(t *testing.T) {
	service, ownerID := setupTestService()
	handler := NewHandlers(service)

	t.Run("CreateAndListSupplements", func(t *testing.T) {
		reqBody := CreateSupplementRequest{
			ProfileID: ownerID,
			Name:      "Витамин D3",
			Components: []ComponentInput{
				{
					NutrientKey:  "vitamin_d",
					HKIdentifier: strPtr("dietaryVitaminD"),
					Amount:       2000,
					Unit:         "IU",
				},
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/v1/supplements", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.HandleCreateSupplement(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		var created SupplementDTO
		json.NewDecoder(w.Body).Decode(&created)

		if created.Name != "Витамин D3" {
			t.Errorf("expected name 'Витамин D3', got '%s'", created.Name)
		}

		if len(created.Components) != 1 {
			t.Errorf("expected 1 component, got %d", len(created.Components))
		}

		// List supplements
		listReq := httptest.NewRequest("GET", "/v1/supplements?profile_id="+ownerID.String(), nil)
		listW := httptest.NewRecorder()

		handler.HandleListSupplements(listW, listReq)

		if listW.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", listW.Code)
		}

		var listResp SupplementsResponse
		json.NewDecoder(listW.Body).Decode(&listResp)

		if len(listResp.Supplements) == 0 {
			t.Errorf("expected supplements list not empty")
		}
	})
}

func TestWaterIntakes(t *testing.T) {
	service, ownerID := setupTestService()
	handler := NewHandlers(service)

	t.Run("AddWaterAndGetDaily", func(t *testing.T) {
		// Add water
		now := time.Now().UTC()
		reqBody := AddWaterRequest{
			ProfileID: ownerID,
			TakenAt:   now,
			AmountMl:  250,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/v1/intakes/water", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.HandleAddWater(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Get daily
		date := now.Format("2006-01-02")
		getDailyReq := httptest.NewRequest("GET", "/v1/intakes/daily?profile_id="+ownerID.String()+"&date="+date, nil)
		getDailyW := httptest.NewRecorder()

		handler.HandleGetIntakesDaily(getDailyW, getDailyReq)

		if getDailyW.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", getDailyW.Code)
		}

		var dailyResp IntakesDailyResponse
		json.NewDecoder(getDailyW.Body).Decode(&dailyResp)

		if dailyResp.WaterTotalMl != 250 {
			t.Errorf("expected water_total_ml=250, got %d", dailyResp.WaterTotalMl)
		}

		if len(dailyResp.WaterEntries) == 0 {
			t.Errorf("expected water entries not empty")
		}
	})

	t.Run("DailyWaterLimitExceeded", func(t *testing.T) {
		now := time.Now().UTC()
		reqBody := AddWaterRequest{
			ProfileID: ownerID,
			TakenAt:   now,
			AmountMl:  9000, // Exceeds limit
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/v1/intakes/water", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.HandleAddWater(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 for limit exceeded, got %d", w.Code)
		}
	})
}

func TestSupplementIntakes(t *testing.T) {
	service, ownerID := setupTestService()
	handler := NewHandlers(service)

	// Create a supplement first
	supplement := CreateSupplementRequest{
		ProfileID: ownerID,
		Name:      "Магний",
	}
	suppBody, _ := json.Marshal(supplement)
	suppReq := httptest.NewRequest("POST", "/v1/supplements", bytes.NewReader(suppBody))
	suppW := httptest.NewRecorder()
	handler.HandleCreateSupplement(suppW, suppReq)

	var created SupplementDTO
	json.NewDecoder(suppW.Body).Decode(&created)
	supplementID := created.ID

	t.Run("UpsertSupplementIntake", func(t *testing.T) {
		date := time.Now().Format("2006-01-02")
		reqBody := UpsertSupplementIntakeRequest{
			ProfileID:    ownerID,
			SupplementID: supplementID,
			Date:         date,
			Status:       "taken",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/v1/intakes/supplements", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.HandleUpsertSupplementIntake(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Get daily - should show status "taken"
		getDailyReq := httptest.NewRequest("GET", "/v1/intakes/daily?profile_id="+ownerID.String()+"&date="+date, nil)
		getDailyW := httptest.NewRecorder()

		handler.HandleGetIntakesDaily(getDailyW, getDailyReq)

		if getDailyW.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", getDailyW.Code)
		}

		var dailyResp IntakesDailyResponse
		json.NewDecoder(getDailyW.Body).Decode(&dailyResp)

		if len(dailyResp.Supplements) == 0 {
			t.Fatalf("expected supplements not empty")
		}

		found := false
		for _, s := range dailyResp.Supplements {
			if s.SupplementID == supplementID && s.Status == "taken" {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("expected supplement status 'taken'")
		}
	})

	t.Run("UpsertTwice_UpdatesStatus", func(t *testing.T) {
		date := time.Now().Format("2006-01-02")

		// First: taken
		reqBody1 := UpsertSupplementIntakeRequest{
			ProfileID:    ownerID,
			SupplementID: supplementID,
			Date:         date,
			Status:       "taken",
		}
		body1, _ := json.Marshal(reqBody1)
		req1 := httptest.NewRequest("POST", "/v1/intakes/supplements", bytes.NewReader(body1))
		w1 := httptest.NewRecorder()
		handler.HandleUpsertSupplementIntake(w1, req1)

		if w1.Code != http.StatusCreated {
			t.Fatalf("expected status 201 for first upsert, got %d", w1.Code)
		}

		// Second: skipped (same day)
		reqBody2 := UpsertSupplementIntakeRequest{
			ProfileID:    ownerID,
			SupplementID: supplementID,
			Date:         date,
			Status:       "skipped",
		}
		body2, _ := json.Marshal(reqBody2)
		req2 := httptest.NewRequest("POST", "/v1/intakes/supplements", bytes.NewReader(body2))
		w2 := httptest.NewRecorder()
		handler.HandleUpsertSupplementIntake(w2, req2)

		if w2.Code != http.StatusCreated {
			t.Fatalf("expected status 201 for second upsert, got %d", w2.Code)
		}

		// Verify status is "skipped" via API (not duplicate)
		getDailyReq := httptest.NewRequest("GET", "/v1/intakes/daily?profile_id="+ownerID.String()+"&date="+date, nil)
		getDailyW := httptest.NewRecorder()

		handler.HandleGetIntakesDaily(getDailyW, getDailyReq)

		if getDailyW.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", getDailyW.Code)
		}

		var dailyResp IntakesDailyResponse
		json.NewDecoder(getDailyW.Body).Decode(&dailyResp)

		found := false
		for _, s := range dailyResp.Supplements {
			if s.SupplementID == supplementID {
				found = true
				if s.Status != "skipped" {
					t.Errorf("expected status 'skipped' after second upsert, got '%s'", s.Status)
				}
				break
			}
		}

		if !found {
			t.Errorf("supplement not found in daily response")
		}
	})
}

func TestIntakesDaily_Empty(t *testing.T) {
	service, ownerID := setupTestService()
	handler := NewHandlers(service)

	date := "2026-01-01"
	req := httptest.NewRequest("GET", "/v1/intakes/daily?profile_id="+ownerID.String()+"&date="+date, nil)
	w := httptest.NewRecorder()

	handler.HandleGetIntakesDaily(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp IntakesDailyResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.WaterTotalMl != 0 {
		t.Errorf("expected water_total_ml=0, got %d", resp.WaterTotalMl)
	}

	if len(resp.Supplements) != 0 {
		t.Errorf("expected supplements empty for profile with no supplements")
	}
}

func strPtr(s string) *string {
	return &s
}
