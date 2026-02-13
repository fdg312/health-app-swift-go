package checkins

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

// mockStorage implements Storage for testing
type mockStorage struct {
	checkins map[uuid.UUID]Checkin
	byKey    map[string]uuid.UUID
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		checkins: make(map[uuid.UUID]Checkin),
		byKey:    make(map[string]uuid.UUID),
	}
}

func (m *mockStorage) ListCheckins(profileID uuid.UUID, from, to string) ([]Checkin, error) {
	var result []Checkin
	for _, c := range m.checkins {
		if c.ProfileID == profileID && c.Date >= from && c.Date <= to {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *mockStorage) GetCheckin(id uuid.UUID) (*Checkin, error) {
	c, exists := m.checkins[id]
	if !exists {
		return nil, ErrCheckinNotFound
	}
	return &c, nil
}

func (m *mockStorage) UpsertCheckin(checkin *Checkin) error {
	key := checkin.ProfileID.String() + ":" + checkin.Date + ":" + checkin.Type
	if existingID, exists := m.byKey[key]; exists {
		existing := m.checkins[existingID]
		existing.Score = checkin.Score
		existing.Tags = checkin.Tags
		existing.Note = checkin.Note
		existing.UpdatedAt = checkin.UpdatedAt
		m.checkins[existingID] = existing
		*checkin = existing
	} else {
		m.checkins[checkin.ID] = *checkin
		m.byKey[key] = checkin.ID
	}
	return nil
}

func (m *mockStorage) DeleteCheckin(id uuid.UUID) error {
	c, exists := m.checkins[id]
	if !exists {
		return ErrCheckinNotFound
	}
	key := c.ProfileID.String() + ":" + c.Date + ":" + c.Type
	delete(m.checkins, id)
	delete(m.byKey, key)
	return nil
}

// mockProfileStorage implements ProfileStorage for testing
type mockProfileStorage struct {
	profiles map[uuid.UUID]storage.Profile
}

func newMockProfileStorage() *mockProfileStorage {
	ownerID := uuid.New()
	return &mockProfileStorage{
		profiles: map[uuid.UUID]storage.Profile{
			ownerID: {
				ID:          ownerID,
				OwnerUserID: "default",
				Type:        "owner",
			},
		},
	}
}

func (m *mockProfileStorage) GetProfile(ctx context.Context, id uuid.UUID) (*storage.Profile, error) {
	p, exists := m.profiles[id]
	if !exists {
		return nil, ErrProfileNotFound
	}
	return &p, nil
}

func (m *mockProfileStorage) AddProfile(id uuid.UUID) {
	m.profiles[id] = storage.Profile{
		ID:          id,
		OwnerUserID: "default",
		Type:        "guest",
	}
}

func TestHandleList(t *testing.T) {
	// Setup
	storage := newMockStorage()
	profileStorage := newMockProfileStorage()
	service := NewService(storage, profileStorage)

	// Get owner profile ID
	var ownerID uuid.UUID
	for id := range profileStorage.profiles {
		ownerID = id
		break
	}

	// Insert a test checkin
	checkin := &Checkin{
		ID:        uuid.New(),
		ProfileID: ownerID,
		Date:      "2026-02-12",
		Type:      TypeMorning,
		Score:     4,
		Tags:      []string{},
		Note:      "test note",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	storage.UpsertCheckin(checkin)

	// Test successful list
	req := httptest.NewRequest("GET", "/v1/checkins?profile_id="+ownerID.String()+"&from=2026-02-01&to=2026-02-28", nil)
	w := httptest.NewRecorder()

	HandleList(service)(w, req.WithContext(context.Background()))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp CheckinsResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Checkins) != 1 {
		t.Errorf("expected 1 checkin, got %d", len(resp.Checkins))
	}
}

func TestHandleUpsert(t *testing.T) {
	// Setup
	storage := newMockStorage()
	profileStorage := newMockProfileStorage()
	service := NewService(storage, profileStorage)

	// Get owner profile ID
	var ownerID uuid.UUID
	for id := range profileStorage.profiles {
		ownerID = id
		break
	}

	// Test successful upsert
	reqBody := UpsertCheckinRequest{
		ProfileID: ownerID,
		Date:      "2026-02-12",
		Type:      TypeEvening,
		Score:     2,
		Tags:      []string{"стресс", "усталость"},
		Note:      "тяжелый день",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/checkins", bytes.NewReader(body))
	w := httptest.NewRecorder()

	HandleUpsert(service)(w, req.WithContext(context.Background()))

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp CheckinDTO
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Score != 2 {
		t.Errorf("expected score 2, got %d", resp.Score)
	}

	if len(resp.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(resp.Tags))
	}
}

func TestHandleUpsertInvalidScore(t *testing.T) {
	// Setup
	storage := newMockStorage()
	profileStorage := newMockProfileStorage()
	service := NewService(storage, profileStorage)

	// Get owner profile ID
	var ownerID uuid.UUID
	for id := range profileStorage.profiles {
		ownerID = id
		break
	}

	// Test invalid score
	reqBody := UpsertCheckinRequest{
		ProfileID: ownerID,
		Date:      "2026-02-12",
		Type:      TypeMorning,
		Score:     6, // Invalid: > 5
		Tags:      []string{},
		Note:      "",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/checkins", bytes.NewReader(body))
	w := httptest.NewRecorder()

	HandleUpsert(service)(w, req.WithContext(context.Background()))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleDelete(t *testing.T) {
	// Setup
	storage := newMockStorage()
	profileStorage := newMockProfileStorage()
	service := NewService(storage, profileStorage)

	// Get owner profile ID
	var ownerID uuid.UUID
	for id := range profileStorage.profiles {
		ownerID = id
		break
	}

	// Insert a test checkin
	checkin := &Checkin{
		ID:        uuid.New(),
		ProfileID: ownerID,
		Date:      "2026-02-12",
		Type:      TypeMorning,
		Score:     3,
		Tags:      []string{},
		Note:      "",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	storage.UpsertCheckin(checkin)

	// Test successful delete
	req := httptest.NewRequest("DELETE", "/v1/checkins/"+checkin.ID.String(), nil)
	req.SetPathValue("id", checkin.ID.String())
	w := httptest.NewRecorder()

	HandleDelete(service)(w, req.WithContext(context.Background()))

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}

	// Verify deleted
	_, err := storage.GetCheckin(checkin.ID)
	if err == nil {
		t.Error("expected checkin to be deleted")
	}
}

func TestHandleDeleteNotFound(t *testing.T) {
	// Setup
	storage := newMockStorage()
	profileStorage := newMockProfileStorage()
	service := NewService(storage, profileStorage)

	// Test delete non-existent checkin
	randomID := uuid.New()
	req := httptest.NewRequest("DELETE", "/v1/checkins/"+randomID.String(), nil)
	req.SetPathValue("id", randomID.String())
	w := httptest.NewRecorder()

	HandleDelete(service)(w, req.WithContext(context.Background()))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// Suppress unused import warnings
var _ = context.Background()
