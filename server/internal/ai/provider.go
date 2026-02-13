package ai

import (
	"context"
	"time"

	"github.com/fdg312/health-hub/internal/settings"
	"github.com/google/uuid"
)

type Provider interface {
	Reply(ctx context.Context, req ReplyRequest) (ReplyResponse, error)
}

type ChatMessage struct {
	Role      string
	Content   string
	CreatedAt time.Time
}

type DaySnapshot struct {
	Date             string
	Steps            int
	ActiveEnergyKcal int
	SleepMinutes     int
	MorningScore     *int
	EveningScore     *int
	NutritionKcal    int
	ProteinG         int
	FatG             int
	CarbsG           int
}

type ReplyRequest struct {
	UserID    string
	ProfileID uuid.UUID
	Messages  []ChatMessage
	Snapshot  DaySnapshot
	Settings  settings.SettingsDTO
	TimeZone  string
}

type ReplyResponse struct {
	AssistantText string
	Proposals     []ProposalDraft
}

type ProposalDraft struct {
	Kind    string
	Title   string
	Summary string
	Payload map[string]any
}
