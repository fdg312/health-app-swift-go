package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fdg312/health-hub/internal/ai"
	"github.com/fdg312/health-hub/internal/config"
	"github.com/fdg312/health-hub/internal/feed"
	"github.com/fdg312/health-hub/internal/settings"
	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/storage/memory"
	"github.com/fdg312/health-hub/internal/userctx"
	"github.com/google/uuid"
)

type mockDaySummaryProvider struct{}

func (m mockDaySummaryProvider) GetDaySummary(ctx context.Context, profileID uuid.UUID, date string) (*feed.FeedDayResponse, error) {
	_ = ctx
	_ = profileID
	return &feed.FeedDayResponse{
		Date: date,
		Daily: map[string]any{
			"activity": map[string]any{
				"steps":              7000,
				"active_energy_kcal": 300,
			},
			"sleep": map[string]any{
				"total_minutes": 430,
			},
		},
	}, nil
}

func TestSendMessageStoresUserAndAssistantMessages(t *testing.T) {
	handler, mem, profileA, _ := setupChatHandler(t)

	reqBody := SendMessageRequest{
		ProfileID: profileA,
		Content:   "Привет, как у меня дела?",
	}
	data, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/messages", bytes.NewReader(data))
	req = req.WithContext(userctx.WithUserID(context.Background(), "userA"))
	w := httptest.NewRecorder()
	handler.HandleSendMessage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp SendMessageResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.AssistantMessage.Role != "assistant" {
		t.Fatalf("expected assistant role, got %q", resp.AssistantMessage.Role)
	}

	rows, _, err := mem.ListMessages(context.Background(), "userA", profileA, 50, nil)
	if err != nil {
		t.Fatalf("list messages failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 stored messages, got %d", len(rows))
	}
	if rows[0].Role != "user" || rows[1].Role != "assistant" {
		t.Fatalf("expected [user, assistant], got [%s, %s]", rows[0].Role, rows[1].Role)
	}
}

func TestListMessagesReturnsHistory(t *testing.T) {
	handler, _, profileA, _ := setupChatHandler(t)

	sendReqBody := SendMessageRequest{
		ProfileID: profileA,
		Content:   "Что по шагам?",
	}
	sendData, _ := json.Marshal(sendReqBody)
	sendReq := httptest.NewRequest(http.MethodPost, "/v1/chat/messages", bytes.NewReader(sendData))
	sendReq = sendReq.WithContext(userctx.WithUserID(context.Background(), "userA"))
	sendW := httptest.NewRecorder()
	handler.HandleSendMessage(sendW, sendReq)
	if sendW.Code != http.StatusOK {
		t.Fatalf("send message failed status=%d body=%s", sendW.Code, sendW.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/chat/messages?profile_id="+profileA.String()+"&limit=50", nil)
	listReq = listReq.WithContext(userctx.WithUserID(context.Background(), "userA"))
	listW := httptest.NewRecorder()
	handler.HandleListMessages(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", listW.Code, listW.Body.String())
	}

	var resp ListMessagesResponse
	if err := json.NewDecoder(listW.Body).Decode(&resp); err != nil {
		t.Fatalf("decode list response failed: %v", err)
	}
	if len(resp.Messages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(resp.Messages))
	}
}

func TestOwnershipForbiddenProfileReturns404(t *testing.T) {
	handler, _, profileA, _ := setupChatHandler(t)

	reqBody := SendMessageRequest{
		ProfileID: profileA,
		Content:   "Попытка доступа к чужому профилю",
	}
	data, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/messages", bytes.NewReader(data))
	req = req.WithContext(userctx.WithUserID(context.Background(), "userB"))
	w := httptest.NewRecorder()
	handler.HandleSendMessage(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestProposalsSavedAndReturned(t *testing.T) {
	handler, mem, profileA, _ := setupChatHandler(t)

	reqBody := SendMessageRequest{
		ProfileID: profileA,
		Content:   "Давай обновим порог шагов и сна",
	}
	data, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/messages", bytes.NewReader(data))
	req = req.WithContext(userctx.WithUserID(context.Background(), "userA"))
	w := httptest.NewRecorder()
	handler.HandleSendMessage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp SendMessageResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(resp.Proposals) == 0 {
		t.Fatalf("expected proposals in response")
	}

	rows, err := mem.List(context.Background(), "userA", profileA, "", 20)
	if err != nil {
		t.Fatalf("list proposals failed: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("expected proposals to be persisted")
	}
}

func setupChatHandler(t *testing.T) (*Handler, *memory.MemoryStorage, uuid.UUID, uuid.UUID) {
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

	service := NewService(
		mem.GetChatStorage(),
		mem.GetProposalsStorage(),
		mem,
		mockDaySummaryProvider{},
		settingsService,
		ai.NewMockProvider(),
	)

	return NewHandler(service), mem, profileA, profileB
}
