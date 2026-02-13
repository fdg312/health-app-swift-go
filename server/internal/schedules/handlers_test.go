package schedules

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/storage/memory"
	"github.com/fdg312/health-hub/internal/userctx"
	"github.com/google/uuid"
)

func TestSchedulesCRUDAndOwnership(t *testing.T) {
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

	supplement := &storage.Supplement{ProfileID: profileA, Name: "Магний"}
	if err := mem.CreateSupplement(ctx, supplement); err != nil {
		t.Fatalf("create supplement: %v", err)
	}

	h := NewHandlers(NewService(mem.GetSupplementSchedulesStorage(), mem.GetSupplementsStorage(), mem))

	// upsert
	upsertBody, _ := json.Marshal(UpsertScheduleRequest{
		ProfileID:    profileA,
		SupplementID: supplement.ID,
		TimeMinutes:  720,
		DaysMask:     127,
		IsEnabled:    true,
	})
	upsertReq := httptest.NewRequest(http.MethodPost, "/v1/schedules/supplements", bytes.NewReader(upsertBody))
	upsertReq = upsertReq.WithContext(userctx.WithUserID(context.Background(), "userA"))
	upsertW := httptest.NewRecorder()
	h.HandleUpsert(upsertW, upsertReq)
	if upsertW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", upsertW.Code, upsertW.Body.String())
	}

	var created ScheduleDTO
	if err := json.NewDecoder(upsertW.Body).Decode(&created); err != nil {
		t.Fatalf("decode upsert: %v", err)
	}
	if created.TimeMinutes != 720 {
		t.Fatalf("expected time=720, got %d", created.TimeMinutes)
	}

	// list
	listReq := httptest.NewRequest(http.MethodGet, "/v1/schedules/supplements?profile_id="+profileA.String(), nil)
	listReq = listReq.WithContext(userctx.WithUserID(context.Background(), "userA"))
	listW := httptest.NewRecorder()
	h.HandleList(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", listW.Code, listW.Body.String())
	}
	var listResp ListSchedulesResponse
	if err := json.NewDecoder(listW.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listResp.Schedules) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(listResp.Schedules))
	}

	// ownership isolation
	crossReq := httptest.NewRequest(http.MethodGet, "/v1/schedules/supplements?profile_id="+profileA.String(), nil)
	crossReq = crossReq.WithContext(userctx.WithUserID(context.Background(), "userB"))
	crossW := httptest.NewRecorder()
	h.HandleList(crossW, crossReq)
	if crossW.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", crossW.Code, crossW.Body.String())
	}

	// delete
	delReq := httptest.NewRequest(http.MethodDelete, "/v1/schedules/supplements/"+created.ID.String(), nil)
	delReq.SetPathValue("id", created.ID.String())
	delReq = delReq.WithContext(userctx.WithUserID(context.Background(), "userA"))
	delW := httptest.NewRecorder()
	h.HandleDelete(delW, delReq)
	if delW.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", delW.Code, delW.Body.String())
	}
}
