package metrics

import (
	"time"

	"github.com/google/uuid"
)

// SyncBatchRequest — запрос для батчевой синхронизации
type SyncBatchRequest struct {
	ProfileID      uuid.UUID        `json:"profile_id"`
	ClientTimeZone string           `json:"client_time_zone,omitempty"`
	Daily          []DailyAggregate `json:"daily,omitempty"`
	Hourly         []HourlyBucket   `json:"hourly,omitempty"`
	Sessions       Sessions         `json:"sessions,omitempty"`
}

// SyncBatchResponse — ответ на батчевую синхронизацию
type SyncBatchResponse struct {
	Status              string `json:"status"`
	UpsertedDaily       int    `json:"upserted_daily"`
	UpsertedHourly      int    `json:"upserted_hourly"`
	InsertedSleepSegs   int    `json:"inserted_sleep_segments"`
	InsertedWorkouts    int    `json:"inserted_workouts"`
}

// DailyAggregate — агрегированные данные за день
type DailyAggregate struct {
	Date        string             `json:"date"` // YYYY-MM-DD
	Sleep       *SleepDaily        `json:"sleep,omitempty"`
	Activity    *ActivityDaily     `json:"activity,omitempty"`
	Body        *BodyDaily         `json:"body,omitempty"`
	Heart       *HeartDaily        `json:"heart,omitempty"`
	Nutrition   *NutritionDaily    `json:"nutrition,omitempty"`
	Intakes     *IntakesDaily      `json:"intakes,omitempty"`
	Temperature *TemperatureDaily  `json:"temperature,omitempty"`
}

type SleepDaily struct {
	TotalMinutes int          `json:"total_minutes"`
	Stages       *SleepStages `json:"stages,omitempty"`
}

type SleepStages struct {
	Rem   int `json:"rem"`
	Deep  int `json:"deep"`
	Core  int `json:"core"`
	Awake int `json:"awake"`
}

type ActivityDaily struct {
	Steps           int     `json:"steps"`
	ActiveEnergyKcal int    `json:"active_energy_kcal"`
	ExerciseMin     int     `json:"exercise_min"`
	StandHours      int     `json:"stand_hours"`
	DistanceKm      float64 `json:"distance_km"`
}

type BodyDaily struct {
	WeightKgLast float64  `json:"weight_kg_last"`
	BMI          float64  `json:"bmi"`
	BodyFatPct   *float64 `json:"body_fat_pct,omitempty"`
}

type HeartDaily struct {
	RestingHrBpm int `json:"resting_hr_bpm"`
}

type NutritionDaily struct {
	EnergyKcal int `json:"energy_kcal"`
	ProteinG   int `json:"protein_g"`
	FatG       int `json:"fat_g"`
	CarbsG     int `json:"carbs_g"`
	CalciumMg  int `json:"calcium_mg"`
}

type IntakesDaily struct {
	WaterMl       int      `json:"water_ml"`
	VitaminsTaken []string `json:"vitamins_taken,omitempty"`
}

type TemperatureDaily struct {
	WristCAvg *float64 `json:"wrist_c_avg,omitempty"`
	WristCMin *float64 `json:"wrist_c_min,omitempty"`
	WristCMax *float64 `json:"wrist_c_max,omitempty"`
}

// HourlyBucket — данные за час
type HourlyBucket struct {
	Hour  time.Time `json:"hour"` // RFC3339, начало часа UTC
	Steps *int      `json:"steps,omitempty"`
	HR    *HRData   `json:"hr,omitempty"`
}

type HRData struct {
	Min int `json:"min"`
	Max int `json:"max"`
	Avg int `json:"avg"`
}

// Sessions — сессии (сон и тренировки)
type Sessions struct {
	SleepSegments []SleepSegment   `json:"sleep_segments,omitempty"`
	Workouts      []WorkoutSession `json:"workouts,omitempty"`
}

// SleepSegment — сегмент сна
type SleepSegment struct {
	Start time.Time `json:"start"` // RFC3339
	End   time.Time `json:"end"`   // RFC3339
	Stage string    `json:"stage"` // "rem"|"deep"|"core"|"awake"
}

// WorkoutSession — сессия тренировки
type WorkoutSession struct {
	Start        time.Time `json:"start"`        // RFC3339
	End          time.Time `json:"end"`          // RFC3339
	Label        string    `json:"label"`        // "strength"|"run"|...
	CaloriesKcal *int      `json:"calories_kcal,omitempty"`
}

// DailyMetricsResponse — ответ для GET /v1/metrics/daily
type DailyMetricsResponse struct {
	Daily []DailyAggregate `json:"daily"`
}

// HourlyMetricsResponse — ответ для GET /v1/metrics/hourly
type HourlyMetricsResponse struct {
	Hourly []HourlyBucket `json:"hourly"`
}

// ErrorResponse — формат ошибки
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
