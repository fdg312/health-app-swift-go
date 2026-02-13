package ai

import (
	"context"
	"fmt"
	"strings"
)

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (p *MockProvider) Reply(ctx context.Context, req ReplyRequest) (ReplyResponse, error) {
	_ = ctx

	lastUserMessage := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			lastUserMessage = req.Messages[i].Content
			break
		}
	}

	text := fmt.Sprintf(
		"Mock-ответ: сегодня шаги %d, активная энергия %d ккал, сон %d минут. %s",
		req.Snapshot.Steps,
		req.Snapshot.ActiveEnergyKcal,
		req.Snapshot.SleepMinutes,
		"Это демо-режим, рекомендации не являются медицинским заключением.",
	)

	proposals := make([]ProposalDraft, 0, 1)
	lowered := strings.ToLower(lastUserMessage)
	if strings.Contains(lowered, "витамин") ||
		strings.Contains(lowered, "добавк") ||
		strings.Contains(lowered, "расписани") {
		proposals = append(proposals, ProposalDraft{
			Kind:    "vitamins_schedule",
			Title:   "Предложение расписания витаминов",
			Summary: "Подготовил расписание приёма добавок. Применение вручную.",
			Payload: map[string]any{
				"replace": true,
				"items": []map[string]any{
					{
						"supplement_name": "Магний",
						"time_minutes":    720,
						"days_mask":       127,
						"is_enabled":      true,
					},
					{
						"supplement_name": "Витамин D",
						"time_minutes":    600,
						"days_mask":       62,
						"is_enabled":      true,
					},
				},
			},
		})
	} else if strings.Contains(lowered, "порог") ||
		strings.Contains(lowered, "шаг") ||
		strings.Contains(lowered, "сон") ||
		strings.Contains(lowered, "уведом") ||
		strings.Contains(lowered, "тихо") {
		proposals = append(proposals, ProposalDraft{
			Kind:    "settings_update",
			Title:   "Подкорректировать пороги активности",
			Summary: "Предлагаю обновить пороги шагов и сна. Изменения применяются только вручную.",
			Payload: map[string]any{
				"min_steps":         8000,
				"min_sleep_minutes": 450,
			},
		})
	} else if strings.Contains(lowered, "тренир") ||
		strings.Contains(lowered, "тренировк") ||
		strings.Contains(lowered, "план") ||
		strings.Contains(lowered, "бег") ||
		strings.Contains(lowered, "силов") ||
		strings.Contains(lowered, "выносл") {
		proposals = append(proposals, ProposalDraft{
			Kind:    "workout_plan",
			Title:   "План тренировок на неделю",
			Summary: "Подготовил сбалансированный план тренировок. Применение вручную.",
			Payload: map[string]any{
				"replace": true,
				"title":   "План выносливости",
				"goal":    "выносливость",
				"items": []map[string]any{
					{
						"kind":         "run",
						"time_minutes": 420,
						"days_mask":    62,
						"duration_min": 30,
						"intensity":    "medium",
						"note":         "лёгкий темп",
						"details":      map[string]any{},
					},
					{
						"kind":         "strength",
						"time_minutes": 1140,
						"days_mask":    20,
						"duration_min": 40,
						"intensity":    "high",
						"note":         "верх тела",
						"details":      map[string]any{},
					},
					{
						"kind":         "core",
						"time_minutes": 480,
						"days_mask":    85,
						"duration_min": 15,
						"intensity":    "medium",
						"note":         "планка и пресс",
						"details":      map[string]any{},
					},
				},
			},
		})
	} else if strings.Contains(lowered, "питани") ||
		strings.Contains(lowered, "ккал") ||
		strings.Contains(lowered, "бжу") ||
		strings.Contains(lowered, "белок") ||
		strings.Contains(lowered, "углевод") ||
		strings.Contains(lowered, "жир") ||
		strings.Contains(lowered, "калори") ||
		strings.Contains(lowered, "диет") {
		proposals = append(proposals, ProposalDraft{
			Kind:    "nutrition_plan",
			Title:   "План питания",
			Summary: "Подготовил рекомендации по целям питания. Применение вручную.",
			Payload: map[string]any{
				"calories_kcal": 2200,
				"protein_g":     120,
				"fat_g":         70,
				"carbs_g":       250,
				"calcium_mg":    800,
			},
		})
	} else if strings.Contains(lowered, "план питания") ||
		strings.Contains(lowered, "еда") ||
		strings.Contains(lowered, "рацион") ||
		strings.Contains(lowered, "меню") ||
		strings.Contains(lowered, "завтрак") ||
		strings.Contains(lowered, "обед") ||
		strings.Contains(lowered, "ужин") {
		proposals = append(proposals, ProposalDraft{
			Kind:    "meal_plan",
			Title:   "План питания на неделю",
			Summary: "Подготовил план питания на 7 дней. Применение вручную.",
			Payload: map[string]any{
				"title": "План питания на неделю",
				"items": []map[string]any{
					{
						"day_index":        0,
						"meal_slot":        "breakfast",
						"title":            "Овсянка с бананом и орехами",
						"notes":            "",
						"approx_kcal":      450,
						"approx_protein_g": 15,
						"approx_fat_g":     12,
						"approx_carbs_g":   70,
					},
					{
						"day_index":        0,
						"meal_slot":        "lunch",
						"title":            "Курица с гречкой и овощами",
						"notes":            "",
						"approx_kcal":      600,
						"approx_protein_g": 45,
						"approx_fat_g":     15,
						"approx_carbs_g":   55,
					},
					{
						"day_index":        0,
						"meal_slot":        "dinner",
						"title":            "Рыба с рисом и салатом",
						"notes":            "",
						"approx_kcal":      550,
						"approx_protein_g": 40,
						"approx_fat_g":     18,
						"approx_carbs_g":   50,
					},
					{
						"day_index":        1,
						"meal_slot":        "breakfast",
						"title":            "Творог с фруктами",
						"notes":            "",
						"approx_kcal":      350,
						"approx_protein_g": 25,
						"approx_fat_g":     8,
						"approx_carbs_g":   45,
					},
					{
						"day_index":        1,
						"meal_slot":        "lunch",
						"title":            "Говядина с картофелем",
						"notes":            "",
						"approx_kcal":      650,
						"approx_protein_g": 50,
						"approx_fat_g":     20,
						"approx_carbs_g":   60,
					},
				},
			},
		})
	}

	return ReplyResponse{
		AssistantText: text,
		Proposals:     proposals,
	}, nil
}
