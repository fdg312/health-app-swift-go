package feed

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/userctx"
	"github.com/google/uuid"
)

var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrInvalidDate     = errors.New("invalid date format")
)

// MetricsStorage defines the interface for metrics operations
type MetricsStorage interface {
	GetDailyMetrics(ctx context.Context, profileID uuid.UUID, from, to string) ([]DailyMetricRow, error)
}

// DailyMetricRow represents a daily metric row
type DailyMetricRow struct {
	ProfileID uuid.UUID
	Date      string
	Payload   []byte
}

// CheckinsStorage defines the interface for checkins operations
type CheckinsStorage interface {
	ListCheckins(ctx context.Context, profileID uuid.UUID, from, to string) ([]Checkin, error)
}

// Checkin represents a checkin
type Checkin struct {
	ID        uuid.UUID
	ProfileID uuid.UUID
	Date      string
	Type      string
	Score     int
	Tags      []string
	Note      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ProfileStorage defines the interface for profile operations
type ProfileStorage interface {
	GetProfile(ctx context.Context, id uuid.UUID) (*storage.Profile, error)
}

// IntakesStorage defines the interface for intakes operations
type IntakesStorage interface {
	GetWaterDaily(ctx context.Context, profileID uuid.UUID, date string) (int, error)
	GetSupplementDaily(ctx context.Context, profileID uuid.UUID, date string) (map[uuid.UUID]string, error)
	ListSupplements(ctx context.Context, profileID uuid.UUID) ([]Supplement, error)
}

// Supplement represents a supplement for counting
type Supplement struct {
	ID uuid.UUID
}

// NutritionTargetsStorage defines the interface for nutrition targets operations
type NutritionTargetsStorage interface {
	Get(ctx context.Context, ownerUserID string, profileID uuid.UUID) (*NutritionTarget, error)
}

// NutritionTarget represents nutrition targets
type NutritionTarget struct {
	CaloriesKcal int
	ProteinG     int
	FatG         int
	CarbsG       int
	CalciumMg    int
}

// MealPlansStorage defines the interface for meal plans operations
type MealPlansStorage interface {
	GetToday(ctx context.Context, ownerUserID string, profileID string, date time.Time) ([]MealPlanItemStorage, error)
	GetActive(ctx context.Context, ownerUserID string, profileID string) (MealPlanStorage, []MealPlanItemStorage, bool, error)
}

// MealPlanStorage represents a meal plan
type MealPlanStorage struct {
	ID    string
	Title string
}

// MealPlanItemStorage represents a meal plan item
type MealPlanItemStorage struct {
	ID             string
	DayIndex       int
	MealSlot       string
	Title          string
	Notes          string
	ApproxKcal     int
	ApproxProteinG int
	ApproxFatG     int
	ApproxCarbsG   int
}

// FoodPrefsStorage defines the interface for food prefs operations
type FoodPrefsStorage interface {
	List(ctx context.Context, ownerUserID string, profileID string, query string, limit, offset int) ([]FoodPref, int, error)
}

// FoodPref represents a food preference
type FoodPref struct {
	ID string
}

// Service handles feed business logic
type Service struct {
	metricsStorage          MetricsStorage
	checkinsStorage         CheckinsStorage
	profileStorage          ProfileStorage
	intakesStorage          IntakesStorage
	nutritionTargetsStorage NutritionTargetsStorage
	mealPlansStorage        MealPlansStorage
	foodPrefsStorage        FoodPrefsStorage
}

// NewService creates a new feed service
func NewService(metricsStorage MetricsStorage, checkinsStorage CheckinsStorage, profileStorage ProfileStorage, intakesStorage IntakesStorage) *Service {
	return &Service{
		metricsStorage:  metricsStorage,
		checkinsStorage: checkinsStorage,
		profileStorage:  profileStorage,
		intakesStorage:  intakesStorage,
	}
}

// WithNutritionTargetsStorage adds nutrition targets storage to the service
func (s *Service) WithNutritionTargetsStorage(storage NutritionTargetsStorage) *Service {
	s.nutritionTargetsStorage = storage
	return s
}

// WithMealPlansStorage adds meal plans storage to the service
func (s *Service) WithMealPlansStorage(storage MealPlansStorage) *Service {
	s.mealPlansStorage = storage
	return s
}

// WithFoodPrefsStorage adds food prefs storage to the service
func (s *Service) WithFoodPrefsStorage(storage FoodPrefsStorage) *Service {
	s.foodPrefsStorage = storage
	return s
}

// GetDaySummary returns a summary for a specific day
func (s *Service) GetDaySummary(ctx context.Context, profileID uuid.UUID, date string) (*FeedDayResponse, error) {
	profile, err := s.profileStorage.GetProfile(ctx, profileID)
	if err != nil {
		return nil, ErrProfileNotFound
	}
	if userID, ok := userctx.GetUserID(ctx); ok && strings.TrimSpace(userID) != "" && profile.OwnerUserID != userID {
		return nil, ErrProfileNotFound
	}

	// Validate date format
	if err := validateDate(date); err != nil {
		return nil, ErrInvalidDate
	}

	missingFields := []string{}

	// Fetch daily metrics
	metricsRows, err := s.metricsStorage.GetDailyMetrics(ctx, profileID, date, date)
	if err != nil {
		return nil, err
	}

	var daily interface{}
	if len(metricsRows) > 0 {
		// Unmarshal payload to generic map to preserve structure
		var payload map[string]interface{}
		if err := json.Unmarshal(metricsRows[0].Payload, &payload); err != nil {
			return nil, err
		}
		daily = payload

		// Check for missing sub-fields in daily metrics
		if payload["body"] != nil {
			if body, ok := payload["body"].(map[string]interface{}); ok {
				if body["weight_kg_last"] == nil || body["weight_kg_last"] == float64(0) {
					missingFields = append(missingFields, "weight")
				}
			}
		} else {
			missingFields = append(missingFields, "weight")
		}

		if payload["heart"] != nil {
			if heart, ok := payload["heart"].(map[string]interface{}); ok {
				if heart["resting_hr_bpm"] == nil || heart["resting_hr_bpm"] == float64(0) {
					missingFields = append(missingFields, "resting_hr")
				}
			}
		} else {
			missingFields = append(missingFields, "resting_hr")
		}
	} else {
		missingFields = append(missingFields, "daily")
	}

	// Fetch checkins
	checkinsRows, err := s.checkinsStorage.ListCheckins(ctx, profileID, date, date)
	if err != nil {
		return nil, err
	}

	checkins := &DayCheckins{}
	hasMorning := false
	hasEvening := false

	for _, c := range checkinsRows {
		summary := &CheckinSummary{
			ID:        c.ID,
			Score:     c.Score,
			Tags:      c.Tags,
			Note:      c.Note,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		}

		if c.Type == "morning" {
			checkins.Morning = summary
			hasMorning = true
		} else if c.Type == "evening" {
			checkins.Evening = summary
			hasEvening = true
		}
	}

	if !hasMorning {
		missingFields = append(missingFields, "morning_checkin")
	}
	if !hasEvening {
		missingFields = append(missingFields, "evening_checkin")
	}

	// Fetch intakes data (optional, no error if not available)
	var intakesSummary *IntakesSummary
	if s.intakesStorage != nil {
		waterTotal, err := s.intakesStorage.GetWaterDaily(ctx, profileID, date)
		if err == nil {
			// Get supplements count
			supplements, err := s.intakesStorage.ListSupplements(ctx, profileID)
			if err == nil {
				supplementStatuses, err := s.intakesStorage.GetSupplementDaily(ctx, profileID, date)
				if err == nil {
					supplementsTaken := 0
					for _, status := range supplementStatuses {
						if status == "taken" {
							supplementsTaken++
						}
					}

					intakesSummary = &IntakesSummary{
						WaterTotalMl:     waterTotal,
						SupplementsTaken: supplementsTaken,
						SupplementsTotal: len(supplements),
					}
				}
			}
		}
	}

	// Fetch nutrition targets (optional, no error if not available or no storage)
	var nutritionTargets *NutritionTargets
	var nutritionProgress *NutritionProgress
	if s.nutritionTargetsStorage != nil {
		target, err := s.nutritionTargetsStorage.Get(ctx, profile.OwnerUserID, profileID)
		if err == nil && target != nil {
			nutritionTargets = &NutritionTargets{
				CaloriesKcal: target.CaloriesKcal,
				ProteinG:     target.ProteinG,
				FatG:         target.FatG,
				CarbsG:       target.CarbsG,
				CalciumMg:    target.CalciumMg,
			}

			// Calculate progress if we have daily metrics with nutrition data
			if daily != nil {
				if dailyMap, ok := daily.(map[string]interface{}); ok {
					if nutritionData, ok := dailyMap["nutrition"].(map[string]interface{}); ok {
						// Extract actual values
						actualCalories := getIntFromMap(nutritionData, "energy_kcal")
						actualProtein := getIntFromMap(nutritionData, "protein_g")
						actualFat := getIntFromMap(nutritionData, "fat_g")
						actualCarbs := getIntFromMap(nutritionData, "carbs_g")
						actualCalcium := getIntFromMap(nutritionData, "calcium_mg")

						// Calculate percentages (capped at 200%)
						nutritionProgress = &NutritionProgress{
							CaloriesKcal:    actualCalories,
							CaloriesPercent: calculatePercent(actualCalories, target.CaloriesKcal),
							ProteinG:        actualProtein,
							ProteinPercent:  calculatePercent(actualProtein, target.ProteinG),
							FatG:            actualFat,
							FatPercent:      calculatePercent(actualFat, target.FatG),
							CarbsG:          actualCarbs,
							CarbsPercent:    calculatePercent(actualCarbs, target.CarbsG),
							CalciumMg:       actualCalcium,
							CalciumPercent:  calculatePercent(actualCalcium, target.CalciumMg),
						}
					}
				}
			}
		}
	}

	// Fetch meal plan for today (optional, no error if not available)
	var mealToday []MealPlanItem
	var mealPlanTitle string
	if s.mealPlansStorage != nil {
		dateTime, err := time.Parse("2006-01-02", date)
		if err == nil {
			items, err := s.mealPlansStorage.GetToday(ctx, profile.OwnerUserID, profileID.String(), dateTime)
			if err == nil && len(items) > 0 {
				// Get plan title from active plan
				plan, _, found, err := s.mealPlansStorage.GetActive(ctx, profile.OwnerUserID, profileID.String())
				if err == nil && found {
					mealPlanTitle = plan.Title
				}

				// Convert storage items to response items
				mealToday = make([]MealPlanItem, len(items))
				for i, item := range items {
					itemID, _ := uuid.Parse(item.ID)
					mealToday[i] = MealPlanItem{
						ID:             itemID,
						DayIndex:       item.DayIndex,
						MealSlot:       item.MealSlot,
						Title:          item.Title,
						Notes:          item.Notes,
						ApproxKcal:     item.ApproxKcal,
						ApproxProteinG: item.ApproxProteinG,
						ApproxFatG:     item.ApproxFatG,
						ApproxCarbsG:   item.ApproxCarbsG,
					}
				}
			}
		}
	}

	// Count food preferences (optional, no error if not available)
	var foodPrefsCount int
	if s.foodPrefsStorage != nil {
		_, total, err := s.foodPrefsStorage.List(ctx, profile.OwnerUserID, profileID.String(), "", 1, 0)
		if err == nil {
			foodPrefsCount = total
		}
	}

	return &FeedDayResponse{
		Date:              date,
		ProfileID:         profileID,
		Daily:             daily,
		Checkins:          checkins,
		Intakes:           intakesSummary,
		NutritionTargets:  nutritionTargets,
		NutritionProgress: nutritionProgress,
		MealToday:         mealToday,
		MealPlanTitle:     mealPlanTitle,
		FoodPrefsCount:    foodPrefsCount,
		MissingFields:     missingFields,
	}, nil
}

func validateDate(date string) error {
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ErrInvalidDate
	}
	return nil
}

// getIntFromMap safely extracts an int from a map[string]interface{}
func getIntFromMap(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		if floatVal, ok := val.(float64); ok {
			return int(floatVal)
		}
		if intVal, ok := val.(int); ok {
			return intVal
		}
	}
	return 0
}

// calculatePercent calculates percentage with cap at 200%
func calculatePercent(actual, target int) int {
	if target == 0 {
		return 0
	}
	percent := (actual * 100) / target
	if percent > 200 {
		return 200
	}
	return percent
}
