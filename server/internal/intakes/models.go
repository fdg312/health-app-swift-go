package intakes

import (
	"time"

	"github.com/google/uuid"
)

// SupplementDTO — DTO для добавки
type SupplementDTO struct {
	ID         uuid.UUID              `json:"id"`
	ProfileID  uuid.UUID              `json:"profile_id"`
	Name       string                 `json:"name"`
	Notes      *string                `json:"notes,omitempty"`
	Components []SupplementComponentDTO `json:"components,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// SupplementComponentDTO — DTO для компонента добавки
type SupplementComponentDTO struct {
	ID           uuid.UUID `json:"id"`
	NutrientKey  string    `json:"nutrient_key"`
	HKIdentifier *string   `json:"hk_identifier,omitempty"`
	Amount       float64   `json:"amount"`
	Unit         string    `json:"unit"`
}

// CreateSupplementRequest — запрос на создание добавки
type CreateSupplementRequest struct {
	ProfileID  uuid.UUID              `json:"profile_id"`
	Name       string                 `json:"name"`
	Notes      *string                `json:"notes,omitempty"`
	Components []ComponentInput       `json:"components,omitempty"`
}

// ComponentInput — компонент для создания/обновления
type ComponentInput struct {
	NutrientKey  string  `json:"nutrient_key"`
	HKIdentifier *string `json:"hk_identifier,omitempty"`
	Amount       float64 `json:"amount"`
	Unit         string  `json:"unit"`
}

// UpdateSupplementRequest — запрос на обновление добавки
type UpdateSupplementRequest struct {
	Name       *string          `json:"name,omitempty"`
	Notes      *string          `json:"notes,omitempty"`
	Components []ComponentInput `json:"components,omitempty"`
}

// SupplementsResponse — ответ со списком добавок
type SupplementsResponse struct {
	Supplements []SupplementDTO `json:"supplements"`
}

// AddWaterRequest — запрос на добавление воды
type AddWaterRequest struct {
	ProfileID uuid.UUID `json:"profile_id"`
	TakenAt   time.Time `json:"taken_at"`
	AmountMl  int       `json:"amount_ml"`
}

// WaterIntakeDTO — DTO для записи о воде
type WaterIntakeDTO struct {
	ID        uuid.UUID `json:"id"`
	TakenAt   time.Time `json:"taken_at"`
	AmountMl  int       `json:"amount_ml"`
	CreatedAt time.Time `json:"created_at"`
}

// UpsertSupplementIntakeRequest — запрос на отметку приёма добавки
type UpsertSupplementIntakeRequest struct {
	ProfileID    uuid.UUID `json:"profile_id"`
	SupplementID uuid.UUID `json:"supplement_id"`
	Date         string    `json:"date"` // YYYY-MM-DD
	Status       string    `json:"status"` // "taken" or "skipped"
}

// IntakesDailyResponse — ответ для GET /v1/intakes/daily
type IntakesDailyResponse struct {
	Date          string                   `json:"date"`
	WaterTotalMl  int                      `json:"water_total_ml"`
	WaterEntries  []WaterIntakeDTO         `json:"water_entries,omitempty"`
	Supplements   []SupplementDailyStatus  `json:"supplements"`
}

// SupplementDailyStatus — статус добавки за день
type SupplementDailyStatus struct {
	SupplementID uuid.UUID `json:"supplement_id"`
	Name         string    `json:"name"`
	Status       string    `json:"status"` // "taken", "skipped", "none"
}

// ErrorResponse — формат ошибки
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
