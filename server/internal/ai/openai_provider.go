package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fdg312/health-hub/internal/config"
)

type OpenAIProvider struct {
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	httpClient  *http.Client
}

func NewOpenAIProvider(cfg *config.Config) *OpenAIProvider {
	timeoutSeconds := cfg.AITimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 20
	}

	return &OpenAIProvider{
		apiKey:      cfg.OpenAIAPIKey,
		model:       cfg.OpenAIModel,
		maxTokens:   cfg.AIMaxOutputTokens,
		temperature: cfg.AITemperature,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}
}

func (p *OpenAIProvider) Reply(ctx context.Context, req ReplyRequest) (ReplyResponse, error) {
	requestPayload := chatCompletionsRequest{
		Model:       p.model,
		Temperature: p.temperature,
		MaxTokens:   p.maxTokens,
		Messages:    p.buildMessages(req),
	}

	body, err := json.Marshal(requestPayload)
	if err != nil {
		return ReplyResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return ReplyResponse{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return ReplyResponse{}, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ReplyResponse{}, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ReplyResponse{}, fmt.Errorf("openai request failed with status %d", resp.StatusCode)
	}

	var parsed chatCompletionsResponse
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return ReplyResponse{}, err
	}
	if len(parsed.Choices) == 0 {
		return ReplyResponse{}, fmt.Errorf("openai response does not contain choices")
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	text, proposals := extractProposalsFromText(content)
	return ReplyResponse{
		AssistantText: text,
		Proposals:     proposals,
	}, nil
}

func (p *OpenAIProvider) buildMessages(req ReplyRequest) []chatMessageRequest {
	messages := make([]chatMessageRequest, 0, len(req.Messages)+2)
	messages = append(messages, chatMessageRequest{
		Role:    "system",
		Content: p.systemPrompt(req),
	})
	for _, msg := range req.Messages {
		role := strings.TrimSpace(msg.Role)
		if role == "" {
			continue
		}
		messages = append(messages, chatMessageRequest{
			Role:    role,
			Content: msg.Content,
		})
	}
	return messages
}

func (p *OpenAIProvider) systemPrompt(req ReplyRequest) string {
	return fmt.Sprintf(
		"Ты помощник HealthHub. Не ставь диагнозы и не заменяй врача. "+
			"Если риск или ухудшение состояния — рекомендуй обратиться к врачу. "+
			"Отвечай кратко и объяснимо, с опорой на метрики пользователя. "+
			"Снимок дня: date=%s, steps=%d, active_energy_kcal=%d, sleep_minutes=%d, nutrition_kcal=%d. "+
			"Если есть полезные структурированные предложения, добавь в конце отдельный блок строго в формате "+
			"<proposals>[{\"kind\":\"settings_update\",\"title\":\"...\",\"summary\":\"...\",\"payload\":{...}}]</proposals>. "+
			"Для kind=settings_update payload должен быть JSON object только с разрешёнными ключами: "+
			"time_zone, quiet_start_minutes, quiet_end_minutes, notifications_max_per_day, "+
			"min_sleep_minutes, min_steps, min_active_energy_kcal, morning_checkin_time_minutes, "+
			"evening_checkin_time_minutes, vitamins_time_minutes. "+
			"Для kind=vitamins_schedule payload должен быть JSON object строго вида: "+
			"{\"replace\":true,\"items\":[{\"supplement_name\":\"...\",\"time_minutes\":720,\"days_mask\":127,\"is_enabled\":true}]}. "+
			"В items не более 20 элементов; supplement_name от 1 до 80 символов; time_minutes 0..1439; days_mask 0..127. "+
			"Для kind=workout_plan payload должен быть JSON object строго вида: "+
			"{\"replace\":true,\"title\":\"...\",\"goal\":\"...\",\"items\":[{\"kind\":\"run\",\"time_minutes\":420,\"days_mask\":62,\"duration_min\":30,\"intensity\":\"medium\",\"note\":\"\",\"details\":{}}]}. "+
			"Для kind=nutrition_plan payload должен быть JSON object строго вида: "+
			"{\"calories_kcal\":2200,\"protein_g\":120,\"fat_g\":70,\"carbs_g\":250,\"calcium_mg\":800}. "+
			"Для kind=meal_plan payload должен быть JSON object строго вида: "+
			"{\"title\":\"План питания на неделю\",\"items\":[{\"day_index\":0,\"meal_slot\":\"breakfast\",\"title\":\"Овсянка\",\"notes\":\"\",\"approx_kcal\":450,\"approx_protein_g\":15,\"approx_fat_g\":12,\"approx_carbs_g\":70}]}. "+
			"В items не более 28 элементов (7 дней × 4 слота); day_index 0..6; meal_slot: breakfast, lunch, dinner, snack. "+
			"Никогда не добавляй profile_id, owner_user_id или любые другие ключи в payload. "+
			"Если предложений нет — не добавляй блок proposals.",
		req.Snapshot.Date,
		req.Snapshot.Steps,
		req.Snapshot.ActiveEnergyKcal,
		req.Snapshot.SleepMinutes,
		req.Snapshot.NutritionKcal,
	)
}

func extractProposalsFromText(content string) (string, []ProposalDraft) {
	startTag := "<proposals>"
	endTag := "</proposals>"
	start := strings.Index(content, startTag)
	end := strings.Index(content, endTag)
	if start == -1 || end == -1 || end <= start {
		return strings.TrimSpace(content), nil
	}

	jsonChunk := strings.TrimSpace(content[start+len(startTag) : end])
	mainText := strings.TrimSpace(content[:start] + content[end+len(endTag):])

	var drafts []ProposalDraft
	if err := json.Unmarshal([]byte(jsonChunk), &drafts); err != nil {
		return mainText, nil
	}

	return mainText, drafts
}

type chatCompletionsRequest struct {
	Model       string               `json:"model"`
	Messages    []chatMessageRequest `json:"messages"`
	Temperature float64              `json:"temperature"`
	MaxTokens   int                  `json:"max_tokens"`
}

type chatMessageRequest struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionsResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}
