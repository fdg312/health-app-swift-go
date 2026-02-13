package proposals

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fdg312/health-hub/internal/config"
	"github.com/fdg312/health-hub/internal/settings"
	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/storage/memory"
	"github.com/fdg312/health-hub/internal/userctx"
	"github.com/google/uuid"
)

func TestListPendingReturnsProposals(t *testing.T) {
	handler, mem, profileA, _, _ := setupProposalsHandler(t)

	pending := createProposal(t, mem, "userA", profileA, "settings_update", []byte(`{"min_steps":8000}`))
	applied := createProposal(t, mem, "userA", profileA, "settings_update", []byte(`{"min_steps":9000}`))
	if err := mem.UpdateStatus(context.Background(), "userA", applied.ID, "applied"); err != nil {
		t.Fatalf("update status failed: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/v1/ai/proposals?profile_id="+profileA.String()+"&status=pending&limit=20",
		nil,
	)
	req = req.WithContext(userctx.WithUserID(context.Background(), "userA"))
	w := httptest.NewRecorder()
	handler.HandleList(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp ListProposalsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode list response failed: %v", err)
	}
	if len(resp.Proposals) != 1 {
		t.Fatalf("expected 1 pending proposal, got %d", len(resp.Proposals))
	}
	if resp.Proposals[0].ID != pending.ID {
		t.Fatalf("unexpected proposal id %s", resp.Proposals[0].ID)
	}
}

func TestApplySettingsUpdateChangesSettings(t *testing.T) {
	handler, mem, profileA, _, settingsService := setupProposalsHandler(t)

	proposal := createProposal(
		t,
		mem,
		"userA",
		profileA,
		"settings_update",
		[]byte(`{"min_steps":8000,"min_sleep_minutes":450}`),
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/proposals/"+proposal.ID.String()+"/apply", nil)
	req.SetPathValue("id", proposal.ID.String())
	req = req.WithContext(userctx.WithUserID(context.Background(), "userA"))
	w := httptest.NewRecorder()
	handler.HandleApply(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp ApplyProposalResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Status != "applied" {
		t.Fatalf("expected status=applied, got %q", resp.Status)
	}

	settingsResp, err := settingsService.GetOrDefault(context.Background(), "userA")
	if err != nil {
		t.Fatalf("get settings failed: %v", err)
	}
	if settingsResp.Settings.MinSteps != 8000 {
		t.Fatalf("expected min_steps=8000, got %d", settingsResp.Settings.MinSteps)
	}
	if settingsResp.Settings.MinSleepMinutes != 450 {
		t.Fatalf("expected min_sleep_minutes=450, got %d", settingsResp.Settings.MinSleepMinutes)
	}

	stored, found, err := mem.Get(context.Background(), "userA", proposal.ID)
	if err != nil {
		t.Fatalf("get proposal failed: %v", err)
	}
	if !found || stored.Status != "applied" {
		t.Fatalf("expected proposal status applied, got found=%v status=%q", found, stored.Status)
	}
}

func TestApplyVitaminsScheduleCreatesSchedules(t *testing.T) {
	handler, mem, profileA, _, _ := setupProposalsHandler(t)

	proposal := createProposal(
		t,
		mem,
		"userA",
		profileA,
		"vitamins_schedule",
		[]byte(`{
			"replace": true,
			"items": [
				{"supplement_name":"Магний","time_minutes":720,"days_mask":127,"is_enabled":true}
			]
		}`),
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/proposals/"+proposal.ID.String()+"/apply", nil)
	req.SetPathValue("id", proposal.ID.String())
	req = req.WithContext(userctx.WithUserID(context.Background(), "userA"))
	w := httptest.NewRecorder()
	handler.HandleApply(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp ApplyProposalResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Status != "applied" {
		t.Fatalf("expected status=applied, got %q", resp.Status)
	}
	if resp.Applied == nil || resp.Applied.SchedulesCreated == nil || *resp.Applied.SchedulesCreated != 1 {
		t.Fatalf("expected applied.schedules_created=1, got %+v", resp.Applied)
	}

	supplements, err := mem.ListSupplements(context.Background(), profileA)
	if err != nil {
		t.Fatalf("list supplements failed: %v", err)
	}
	if len(supplements) != 1 {
		t.Fatalf("expected 1 supplement, got %d", len(supplements))
	}
	if supplements[0].Name != "Магний" {
		t.Fatalf("unexpected supplement name %q", supplements[0].Name)
	}

	schedules, err := mem.ListSchedules(context.Background(), "userA", profileA)
	if err != nil {
		t.Fatalf("list schedules failed: %v", err)
	}
	if len(schedules) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(schedules))
	}
	if schedules[0].TimeMinutes != 720 || schedules[0].DaysMask != 127 {
		t.Fatalf("unexpected schedule payload: %+v", schedules[0])
	}
}

func TestApplyNonPendingReturns409(t *testing.T) {
	handler, mem, profileA, _, _ := setupProposalsHandler(t)

	proposal := createProposal(t, mem, "userA", profileA, "settings_update", []byte(`{"min_steps":8000}`))
	if err := mem.UpdateStatus(context.Background(), "userA", proposal.ID, "applied"); err != nil {
		t.Fatalf("update status failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/proposals/"+proposal.ID.String()+"/apply", nil)
	req.SetPathValue("id", proposal.ID.String())
	req = req.WithContext(userctx.WithUserID(context.Background(), "userA"))
	w := httptest.NewRecorder()
	handler.HandleApply(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRejectPendingSetsRejectedStatus(t *testing.T) {
	handler, mem, profileA, _, _ := setupProposalsHandler(t)

	proposal := createProposal(t, mem, "userA", profileA, "settings_update", []byte(`{"min_steps":8000}`))

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/proposals/"+proposal.ID.String()+"/reject", nil)
	req.SetPathValue("id", proposal.ID.String())
	req = req.WithContext(userctx.WithUserID(context.Background(), "userA"))
	w := httptest.NewRecorder()
	handler.HandleReject(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	stored, found, err := mem.Get(context.Background(), "userA", proposal.ID)
	if err != nil {
		t.Fatalf("get proposal failed: %v", err)
	}
	if !found || stored.Status != "rejected" {
		t.Fatalf("expected proposal status rejected, got found=%v status=%q", found, stored.Status)
	}
}

func TestOwnershipCrossUserReturns404(t *testing.T) {
	handler, mem, profileA, _, _ := setupProposalsHandler(t)

	proposal := createProposal(t, mem, "userA", profileA, "settings_update", []byte(`{"min_steps":8000}`))

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/proposals/"+proposal.ID.String()+"/apply", nil)
	req.SetPathValue("id", proposal.ID.String())
	req = req.WithContext(userctx.WithUserID(context.Background(), "userB"))
	w := httptest.NewRecorder()
	handler.HandleApply(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func setupProposalsHandler(t *testing.T) (*Handler, *memory.MemoryStorage, uuid.UUID, uuid.UUID, *settings.Service) {
	t.Helper()

	mem := memory.New()
	profileA := uuid.New()
	profileB := uuid.New()

	if err := mem.CreateProfile(context.Background(), &storage.Profile{
		ID:          profileA,
		OwnerUserID: "userA",
		Type:        "owner",
		Name:        "User A",
	}); err != nil {
		t.Fatalf("create profile A failed: %v", err)
	}
	if err := mem.CreateProfile(context.Background(), &storage.Profile{
		ID:          profileB,
		OwnerUserID: "userB",
		Type:        "owner",
		Name:        "User B",
	}); err != nil {
		t.Fatalf("create profile B failed: %v", err)
	}

	cfg := &config.Config{
		NotificationsMaxPerDay:     4,
		DefaultSleepMinMinutes:     420,
		DefaultStepsMin:            6000,
		DefaultActiveEnergyMinKcal: 250,
	}
	settingsService := settings.NewService(mem, cfg)

	service := NewService(mem.GetProposalsStorage(), mem, settingsService)
	return NewHandler(service), mem, profileA, profileB, settingsService
}

func createProposal(
	t *testing.T,
	mem *memory.MemoryStorage,
	ownerUserID string,
	profileID uuid.UUID,
	kind string,
	payload []byte,
) storage.AIProposal {
	t.Helper()

	rows, err := mem.InsertMany(context.Background(), ownerUserID, profileID, []storage.ProposalDraft{
		{
			Kind:    kind,
			Title:   "Proposal",
			Summary: "Summary",
			Payload: payload,
		},
	})
	if err != nil {
		t.Fatalf("insert proposal failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 inserted proposal, got %d", len(rows))
	}
	return rows[0]
}
