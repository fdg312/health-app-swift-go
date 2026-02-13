package chat

import (
	"encoding/json"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

type ChatMessageDTO struct {
	ID        uuid.UUID `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type ProposalDTO struct {
	ID        uuid.UUID      `json:"id"`
	ProfileID uuid.UUID      `json:"profile_id"`
	Kind      string         `json:"kind"`
	Title     string         `json:"title"`
	Summary   string         `json:"summary"`
	Status    string         `json:"status"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}

type SendMessageRequest struct {
	ProfileID uuid.UUID `json:"profile_id"`
	Content   string    `json:"content"`
}

type SendMessageResponse struct {
	AssistantMessage ChatMessageDTO `json:"assistant_message"`
	Proposals        []ProposalDTO  `json:"proposals"`
}

type ListMessagesResponse struct {
	Messages   []ChatMessageDTO `json:"messages"`
	NextCursor *string          `json:"next_cursor,omitempty"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func messageToDTO(msg storage.ChatMessage) ChatMessageDTO {
	return ChatMessageDTO{
		ID:        msg.ID,
		Role:      msg.Role,
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt,
	}
}

func proposalToDTO(p storage.AIProposal) ProposalDTO {
	payload := make(map[string]any)
	if len(p.Payload) > 0 {
		_ = json.Unmarshal(p.Payload, &payload)
	}
	return ProposalDTO{
		ID:        p.ID,
		ProfileID: p.ProfileID,
		Kind:      p.Kind,
		Title:     p.Title,
		Summary:   p.Summary,
		Status:    p.Status,
		Payload:   payload,
		CreatedAt: p.CreatedAt,
	}
}
