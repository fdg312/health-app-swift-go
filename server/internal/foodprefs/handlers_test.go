package foodprefs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fdg312/health-hub/internal/userctx"

	"github.com/fdg312/health-hub/internal/storage"
)

type mockFoodPrefsRepo struct {
	prefs      []storage.FoodPref
	upsertFunc func(ctx context.Context, ownerUserID, profileID string, upsert storage.FoodPrefUpsert) (storage.FoodPref, error)
	listFunc   func(ctx context.Context, ownerUserID, profileID, query string, limit, offset int) ([]storage.FoodPref, int, error)
	deleteFunc func(ctx context.Context, ownerUserID, id string) error
}

func (m *mockFoodPrefsRepo) Upsert(ctx context.Context, ownerUserID, profileID string, upsert storage.FoodPrefUpsert) (storage.FoodPref, error) {
	if m.upsertFunc != nil {
		return m.upsertFunc(ctx, ownerUserID, profileID, upsert)
	}

	// Generate ID if not provided
	id := upsert.ID
	if id == "" {
		id = fmt.Sprintf("fp%d", len(m.prefs)+1)
	}

	pref := storage.FoodPref{
		ID:              id,
		OwnerUserID:     ownerUserID,
		ProfileID:       profileID,
		Name:            upsert.Name,
		Tags:            upsert.Tags,
		KcalPer100g:     upsert.KcalPer100g,
		ProteinGPer100g: upsert.ProteinGPer100g,
		FatGPer100g:     upsert.FatGPer100g,
		CarbsGPer100g:   upsert.CarbsGPer100g,
	}
	m.prefs = append(m.prefs, pref)
	return pref, nil
}

func (m *mockFoodPrefsRepo) List(ctx context.Context, ownerUserID, profileID, query string, limit, offset int) ([]storage.FoodPref, int, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, ownerUserID, profileID, query, limit, offset)
	}

	filtered := []storage.FoodPref{}
	for _, p := range m.prefs {
		if p.OwnerUserID == ownerUserID && p.ProfileID == profileID {
			if query == "" || contains(p.Name, query) {
				filtered = append(filtered, p)
			}
		}
	}

	start := offset
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], len(filtered), nil
}

func (m *mockFoodPrefsRepo) Delete(ctx context.Context, ownerUserID, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, ownerUserID, id)
	}

	for i, p := range m.prefs {
		if p.ID == id && p.OwnerUserID == ownerUserID {
			m.prefs = append(m.prefs[:i], m.prefs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("food preference not found or unauthorized")
}

func (m *mockFoodPrefsRepo) Count(ctx context.Context, ownerUserID, profileID string) (int, error) {
	count := 0
	for _, p := range m.prefs {
		if p.OwnerUserID == ownerUserID && p.ProfileID == profileID {
			count++
		}
	}
	return count, nil
}

func (m *mockFoodPrefsRepo) ExistsByName(ctx context.Context, ownerUserID, profileID, name string) (bool, error) {
	for _, p := range m.prefs {
		if p.OwnerUserID == ownerUserID && p.ProfileID == profileID && p.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		bytes.Contains([]byte(s), []byte(substr)))
}

func TestHandleList_CreateAndList(t *testing.T) {
	repo := &mockFoodPrefsRepo{}
	service := NewService(repo)
	handler := NewHandler(service)

	// Create a food pref
	pref := storage.FoodPref{
		ID:              "fp1",
		OwnerUserID:     "user1",
		ProfileID:       "profile1",
		Name:            "Apple",
		Tags:            []string{"fruit", "healthy"},
		KcalPer100g:     52,
		ProteinGPer100g: 0,
		FatGPer100g:     0,
		CarbsGPer100g:   14,
	}
	repo.prefs = []storage.FoodPref{pref}

	// List request
	req := httptest.NewRequest(http.MethodGet, "/v1/food/prefs?profile_id=profile1&limit=50&offset=0", nil)
	ctx := userctx.WithUserID(req.Context(), "user1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleList(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response ListFoodPrefsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(response.Items))
	}

	if response.Items[0].Name != "Apple" {
		t.Errorf("expected name 'Apple', got '%s'", response.Items[0].Name)
	}

	if response.Total != 1 {
		t.Errorf("expected total 1, got %d", response.Total)
	}
}

func TestHandleList_SearchQuery(t *testing.T) {
	repo := &mockFoodPrefsRepo{}
	service := NewService(repo)
	handler := NewHandler(service)

	repo.prefs = []storage.FoodPref{
		{ID: "fp1", OwnerUserID: "user1", ProfileID: "profile1", Name: "Apple", Tags: []string{}},
		{ID: "fp2", OwnerUserID: "user1", ProfileID: "profile1", Name: "Banana", Tags: []string{}},
		{ID: "fp3", OwnerUserID: "user1", ProfileID: "profile1", Name: "Orange", Tags: []string{}},
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/food/prefs?profile_id=profile1&q=an&limit=50&offset=0", nil)
	ctx := userctx.WithUserID(req.Context(), "user1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleList(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response ListFoodPrefsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should match "Banana" and "Orange" (both contain "an")
	if len(response.Items) != 2 {
		t.Errorf("expected 2 items matching 'an', got %d", len(response.Items))
	}
}

func TestHandleDelete_Success(t *testing.T) {
	repo := &mockFoodPrefsRepo{}
	service := NewService(repo)
	handler := NewHandler(service)

	pref := storage.FoodPref{
		ID:          "fp1",
		OwnerUserID: "user1",
		ProfileID:   "profile1",
		Name:        "Apple",
	}
	repo.prefs = []storage.FoodPref{pref}

	req := httptest.NewRequest(http.MethodDelete, "/v1/food/prefs/fp1", nil)
	ctx := userctx.WithUserID(req.Context(), "user1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleDelete(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}

	if len(repo.prefs) != 0 {
		t.Errorf("expected food pref to be deleted, but still exists")
	}
}

func TestHandleDelete_Ownership(t *testing.T) {
	repo := &mockFoodPrefsRepo{}
	service := NewService(repo)
	handler := NewHandler(service)

	// Food pref owned by user1
	pref := storage.FoodPref{
		ID:          "fp1",
		OwnerUserID: "user1",
		ProfileID:   "profile1",
		Name:        "Apple",
	}
	repo.prefs = []storage.FoodPref{pref}

	// Try to delete with user2 (different owner)
	req := httptest.NewRequest(http.MethodDelete, "/v1/food/prefs/fp1", nil)
	ctx := userctx.WithUserID(req.Context(), "user2")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleDelete(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404 (ownership protection), got %d", w.Code)
	}

	// Verify the pref still exists
	if len(repo.prefs) != 1 {
		t.Errorf("food pref should not be deleted by unauthorized user")
	}
}

func TestHandleList_OwnershipProtection(t *testing.T) {
	repo := &mockFoodPrefsRepo{}
	service := NewService(repo)
	handler := NewHandler(service)

	// Create prefs for two different users
	repo.prefs = []storage.FoodPref{
		{ID: "fp1", OwnerUserID: "user1", ProfileID: "profile1", Name: "Apple"},
		{ID: "fp2", OwnerUserID: "user2", ProfileID: "profile2", Name: "Banana"},
	}

	// User1 tries to list - should only see their own
	req := httptest.NewRequest(http.MethodGet, "/v1/food/prefs?profile_id=profile1&limit=50&offset=0", nil)
	ctx := userctx.WithUserID(req.Context(), "user1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleList(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response ListFoodPrefsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Items) != 1 {
		t.Errorf("expected 1 item (only user1's prefs), got %d", len(response.Items))
	}

	if response.Items[0].Name != "Apple" {
		t.Errorf("expected 'Apple', got '%s'", response.Items[0].Name)
	}
}
