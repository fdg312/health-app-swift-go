package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fdg312/health-hub/internal/ai"
	"github.com/fdg312/health-hub/internal/feed"
	"github.com/fdg312/health-hub/internal/settings"
	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/userctx"
	"github.com/google/uuid"
)

var (
	ErrUnauthorized    = errors.New("unauthorized")
	ErrInvalidRequest  = errors.New("invalid request")
	ErrProfileNotFound = errors.New("profile not found")
	ErrAIFailed        = errors.New("ai failed")
)

type settingsProvider interface {
	GetOrDefault(ctx context.Context, ownerUserID string) (settings.SettingsResponse, error)
}

type daySummaryProvider interface {
	GetDaySummary(ctx context.Context, profileID uuid.UUID, date string) (*feed.FeedDayResponse, error)
}

type Service struct {
	chatStorage      storage.ChatStorage
	proposalsStorage storage.ProposalsStorage
	profilesStorage  storage.Storage
	feedService      daySummaryProvider
	settingsService  settingsProvider
	provider         ai.Provider
	now              func() time.Time
}

func NewService(
	chatStorage storage.ChatStorage,
	proposalsStorage storage.ProposalsStorage,
	profilesStorage storage.Storage,
	feedService daySummaryProvider,
	settingsService settingsProvider,
	provider ai.Provider,
) *Service {
	return &Service{
		chatStorage:      chatStorage,
		proposalsStorage: proposalsStorage,
		profilesStorage:  profilesStorage,
		feedService:      feedService,
		settingsService:  settingsService,
		provider:         provider,
		now:              time.Now,
	}
}

func (s *Service) ListMessages(ctx context.Context, profileID uuid.UUID, limit int, before *time.Time) (*ListMessagesResponse, error) {
	userID := strings.TrimSpace(userIDFromContext(ctx))
	if userID == "" {
		return nil, ErrUnauthorized
	}

	if _, err := s.ensureProfileOwned(ctx, userID, profileID); err != nil {
		return nil, err
	}

	limit = normalizeLimit(limit)
	rows, nextCursorTime, err := s.chatStorage.ListMessages(ctx, userID, profileID, limit, before)
	if err != nil {
		return nil, err
	}

	messages := make([]ChatMessageDTO, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, messageToDTO(row))
	}

	var nextCursor *string
	if nextCursorTime != nil {
		cursor := nextCursorTime.UTC().Format(time.RFC3339Nano)
		nextCursor = &cursor
	}

	return &ListMessagesResponse{
		Messages:   messages,
		NextCursor: nextCursor,
	}, nil
}

func (s *Service) SendMessage(ctx context.Context, req SendMessageRequest) (*SendMessageResponse, error) {
	userID := strings.TrimSpace(userIDFromContext(ctx))
	if userID == "" {
		return nil, ErrUnauthorized
	}

	content := strings.TrimSpace(req.Content)
	if req.ProfileID == uuid.Nil || content == "" {
		return nil, ErrInvalidRequest
	}

	if _, err := s.ensureProfileOwned(ctx, userID, req.ProfileID); err != nil {
		return nil, err
	}

	if _, err := s.chatStorage.InsertMessage(ctx, userID, req.ProfileID, "user", content); err != nil {
		return nil, err
	}

	historyRows, _, err := s.chatStorage.ListMessages(ctx, userID, req.ProfileID, 20, nil)
	if err != nil {
		return nil, err
	}

	settingsResp, err := s.settingsService.GetOrDefault(ctx, userID)
	if err != nil {
		return nil, err
	}

	snapshot, tz, err := s.buildSnapshot(ctx, req.ProfileID, settingsResp.Settings)
	if err != nil {
		return nil, err
	}

	aiMessages := make([]ai.ChatMessage, 0, len(historyRows))
	for _, msg := range historyRows {
		aiMessages = append(aiMessages, ai.ChatMessage{
			Role:      msg.Role,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
		})
	}

	reply, err := s.provider.Reply(ctx, ai.ReplyRequest{
		UserID:    userID,
		ProfileID: req.ProfileID,
		Messages:  aiMessages,
		Snapshot:  snapshot,
		Settings:  settingsResp.Settings,
		TimeZone:  tz,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAIFailed, err)
	}

	assistantText := strings.TrimSpace(reply.AssistantText)
	if assistantText == "" {
		assistantText = "Я не смог сформировать ответ. Попробуйте переформулировать вопрос."
	}

	assistantMessage, err := s.chatStorage.InsertMessage(ctx, userID, req.ProfileID, "assistant", assistantText)
	if err != nil {
		return nil, err
	}

	drafts := make([]storage.ProposalDraft, 0, len(reply.Proposals))
	for _, draft := range reply.Proposals {
		payload, err := json.Marshal(draft.Payload)
		if err != nil {
			payload = []byte(`{}`)
		}
		drafts = append(drafts, storage.ProposalDraft{
			Kind:    normalizeProposalKind(draft.Kind),
			Title:   strings.TrimSpace(draft.Title),
			Summary: strings.TrimSpace(draft.Summary),
			Payload: payload,
		})
	}

	savedProposals, err := s.proposalsStorage.InsertMany(ctx, userID, req.ProfileID, drafts)
	if err != nil {
		return nil, err
	}

	proposalDTOs := make([]ProposalDTO, 0, len(savedProposals))
	for _, proposal := range savedProposals {
		proposalDTOs = append(proposalDTOs, proposalToDTO(proposal))
	}

	return &SendMessageResponse{
		AssistantMessage: messageToDTO(assistantMessage),
		Proposals:        proposalDTOs,
	}, nil
}

func (s *Service) buildSnapshot(ctx context.Context, profileID uuid.UUID, userSettings settings.SettingsDTO) (ai.DaySnapshot, string, error) {
	loc := time.UTC
	tz := "UTC"
	if userSettings.TimeZone != nil {
		if loaded, err := time.LoadLocation(strings.TrimSpace(*userSettings.TimeZone)); err == nil {
			loc = loaded
			tz = loaded.String()
		}
	}

	date := s.now().In(loc).Format("2006-01-02")

	snapshot := ai.DaySnapshot{
		Date: date,
	}

	if s.feedService == nil {
		return snapshot, tz, nil
	}

	daySummary, err := s.feedService.GetDaySummary(ctx, profileID, date)
	if err != nil {
		return ai.DaySnapshot{}, tz, err
	}

	if daySummary == nil {
		return snapshot, tz, nil
	}

	if daySummary.Daily != nil {
		if payload, ok := daySummary.Daily.(map[string]any); ok {
			snapshot.Steps = getNestedInt(payload, "activity", "steps")
			snapshot.ActiveEnergyKcal = getNestedInt(payload, "activity", "active_energy_kcal")
			snapshot.SleepMinutes = getNestedInt(payload, "sleep", "total_minutes")
			snapshot.NutritionKcal = getNestedInt(payload, "nutrition", "energy_kcal")
			snapshot.ProteinG = getNestedInt(payload, "nutrition", "protein_g")
			snapshot.FatG = getNestedInt(payload, "nutrition", "fat_g")
			snapshot.CarbsG = getNestedInt(payload, "nutrition", "carbs_g")
		} else {
			encoded, _ := json.Marshal(daySummary.Daily)
			var payload map[string]any
			if err := json.Unmarshal(encoded, &payload); err == nil {
				snapshot.Steps = getNestedInt(payload, "activity", "steps")
				snapshot.ActiveEnergyKcal = getNestedInt(payload, "activity", "active_energy_kcal")
				snapshot.SleepMinutes = getNestedInt(payload, "sleep", "total_minutes")
				snapshot.NutritionKcal = getNestedInt(payload, "nutrition", "energy_kcal")
				snapshot.ProteinG = getNestedInt(payload, "nutrition", "protein_g")
				snapshot.FatG = getNestedInt(payload, "nutrition", "fat_g")
				snapshot.CarbsG = getNestedInt(payload, "nutrition", "carbs_g")
			}
		}
	}

	if daySummary.Checkins != nil {
		if daySummary.Checkins.Morning != nil {
			score := daySummary.Checkins.Morning.Score
			snapshot.MorningScore = &score
		}
		if daySummary.Checkins.Evening != nil {
			score := daySummary.Checkins.Evening.Score
			snapshot.EveningScore = &score
		}
	}

	return snapshot, tz, nil
}

func (s *Service) ensureProfileOwned(ctx context.Context, ownerUserID string, profileID uuid.UUID) (*storage.Profile, error) {
	profile, err := s.profilesStorage.GetProfile(ctx, profileID)
	if err != nil {
		return nil, ErrProfileNotFound
	}
	if profile.OwnerUserID != ownerUserID {
		return nil, ErrProfileNotFound
	}
	return profile, nil
}

func userIDFromContext(ctx context.Context) string {
	userID, ok := userctx.GetUserID(ctx)
	if !ok {
		return ""
	}
	return strings.TrimSpace(userID)
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func normalizeProposalKind(kind string) string {
	normalized := strings.TrimSpace(kind)
	switch normalized {
	case "settings_update", "vitamins_schedule", "workout_plan", "nutrition_plan", "generic":
		return normalized
	default:
		return "generic"
	}
}

func getNestedInt(payload map[string]any, keys ...string) int {
	var current any = payload
	for _, key := range keys {
		obj, ok := current.(map[string]any)
		if !ok {
			return 0
		}
		current, ok = obj[key]
		if !ok {
			return 0
		}
	}
	switch value := current.(type) {
	case float64:
		return int(value)
	case float32:
		return int(value)
	case int:
		return value
	case int64:
		return int(value)
	case json.Number:
		v, _ := value.Int64()
		return int(v)
	default:
		return 0
	}
}
