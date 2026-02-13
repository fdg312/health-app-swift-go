package profiles

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fdg312/health-hub/internal/storage/memory"
	"github.com/google/uuid"
)

func TestHandleList(t *testing.T) {
	store := memory.New()
	service := NewService(store)
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/v1/profiles", nil)
	w := httptest.NewRecorder()

	handler.HandleList(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp ProfilesResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Должен быть один owner профиль по умолчанию
	if len(resp.Profiles) != 1 {
		t.Errorf("expected 1 profile, got %d", len(resp.Profiles))
	}

	if resp.Profiles[0].Type != "owner" {
		t.Errorf("expected owner profile, got %s", resp.Profiles[0].Type)
	}
}

func TestHandleCreate(t *testing.T) {
	store := memory.New()
	service := NewService(store)
	handler := NewHandler(service)

	reqBody := CreateProfileRequest{
		Type: "guest",
		Name: "Guest 1",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/profiles", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleCreate(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var resp ProfileDTO
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Type != "guest" {
		t.Errorf("expected guest profile, got %s", resp.Type)
	}

	if resp.Name != "Guest 1" {
		t.Errorf("expected name 'Guest 1', got %s", resp.Name)
	}
}

func TestHandleCreateInvalidType(t *testing.T) {
	store := memory.New()
	service := NewService(store)
	handler := NewHandler(service)

	reqBody := CreateProfileRequest{
		Type: "owner",
		Name: "Should Fail",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/profiles", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleCreate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error.Code != "invalid_type" {
		t.Errorf("expected error code 'invalid_type', got %s", resp.Error.Code)
	}
}

func TestHandleDelete(t *testing.T) {
	store := memory.New()
	service := NewService(store)
	handler := NewHandler(service)

	// Создаём guest профиль
	guest, _ := service.CreateProfile(testContext(), CreateProfileRequest{
		Type: "guest",
		Name: "Guest to Delete",
	})

	req := httptest.NewRequest(http.MethodDelete, "/v1/profiles/"+guest.ID.String(), nil)
	w := httptest.NewRecorder()

	handler.HandleDelete(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}

	// Проверяем, что профиль удалён
	_, err := service.GetProfile(testContext(), guest.ID)
	if err == nil {
		t.Error("expected profile to be deleted")
	}
}

func TestHandleDeleteOwner(t *testing.T) {
	store := memory.New()
	service := NewService(store)
	handler := NewHandler(service)

	// Получаем owner профиль
	profiles, _ := service.ListProfiles(testContext())
	ownerID := uuid.Nil
	for _, p := range profiles {
		if p.Type == "owner" {
			ownerID = p.ID
			break
		}
	}

	req := httptest.NewRequest(http.MethodDelete, "/v1/profiles/"+ownerID.String(), nil)
	w := httptest.NewRecorder()

	handler.HandleDelete(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error.Code != "cannot_delete_owner" {
		t.Errorf("expected error code 'cannot_delete_owner', got %s", resp.Error.Code)
	}
}

func TestHandleUpdate(t *testing.T) {
	store := memory.New()
	service := NewService(store)
	handler := NewHandler(service)

	// Создаём guest профиль
	guest, _ := service.CreateProfile(testContext(), CreateProfileRequest{
		Type: "guest",
		Name: "Original Name",
	})

	reqBody := UpdateProfileRequest{
		Name: "Updated Name",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPatch, "/v1/profiles/"+guest.ID.String(), bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleUpdate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp ProfileDTO
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %s", resp.Name)
	}
}

func testContext() context.Context {
	return context.Background()
}
