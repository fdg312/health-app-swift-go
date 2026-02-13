package proposals

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/fdg312/health-hub/internal/mealplans"
	"github.com/fdg312/health-hub/internal/settings"
	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/userctx"
	"github.com/fdg312/health-hub/internal/workouts"
	"github.com/google/uuid"
)

var (
	ErrUnauthorized     = errors.New("unauthorized")
	ErrInvalidRequest   = errors.New("invalid request")
	ErrInvalidPayload   = errors.New("invalid payload")
	ErrUnsupportedKind  = errors.New("unsupported kind")
	ErrProposalNotFound = errors.New("proposal not found")
	ErrNotPending       = errors.New("not pending")
)

type settingsService interface {
	GetOrDefault(ctx context.Context, ownerUserID string) (settings.SettingsResponse, error)
	Upsert(ctx context.Context, ownerUserID string, dto settings.SettingsDTO) (settings.SettingsDTO, error)
}

type workoutService interface {
	ReplacePlanAndItems(ctx context.Context, req *workouts.ReplaceItemsRequest) (*workouts.ReplaceItemsResponse, error)
}

type nutritionService interface {
	UpsertSimple(ctx context.Context, ownerUserID string, profileID uuid.UUID, caloriesKcal, proteinG, fatG, carbsG, calciumMg int) error
}

type mealPlanService interface {
	ReplaceActive(ctx context.Context, ownerUserID string, req mealplans.ReplaceMealPlanRequest) (*mealplans.MealPlanDTO, []mealplans.MealPlanItemDTO, error)
}

type Service struct {
	proposalsStorage storage.ProposalsStorage
	profileStorage   storage.Storage
	settingsService  settingsService
	workoutService   workoutService
	nutritionService nutritionService
	mealPlanService  mealPlanService
}

func NewService(
	proposalsStorage storage.ProposalsStorage,
	profileStorage storage.Storage,
	settingsService settingsService,
) *Service {
	return &Service{
		proposalsStorage: proposalsStorage,
		profileStorage:   profileStorage,
		settingsService:  settingsService,
	}
}

// WithWorkoutService adds workout service for workout_plan proposals
func (s *Service) WithWorkoutService(ws workoutService) *Service {
	s.workoutService = ws
	return s
}

// WithNutritionService adds nutrition service for nutrition_plan proposals
func (s *Service) WithNutritionService(ns nutritionService) *Service {
	s.nutritionService = ns
	return s
}

// WithMealPlanService adds meal plan service for meal_plan proposals
func (s *Service) WithMealPlanService(ms mealPlanService) *Service {
	s.mealPlanService = ms
	return s
}

func (s *Service) List(ctx context.Context, profileID uuid.UUID, status string, limit int) (*ListProposalsResponse, error) {
	userID := strings.TrimSpace(userIDFromContext(ctx))
	if userID == "" {
		return nil, ErrUnauthorized
	}
	if profileID == uuid.Nil {
		return nil, ErrInvalidRequest
	}

	statusFilter, err := normalizeStatusFilter(status)
	if err != nil {
		return nil, ErrInvalidRequest
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	if err := s.ensureProfileOwned(ctx, userID, profileID); err != nil {
		return nil, err
	}

	rows, err := s.proposalsStorage.List(ctx, userID, profileID, statusFilter, limit)
	if err != nil {
		return nil, err
	}

	dtos := make([]ProposalDTO, 0, len(rows))
	for _, row := range rows {
		dtos = append(dtos, proposalToDTO(row))
	}

	return &ListProposalsResponse{Proposals: dtos}, nil
}

func (s *Service) Apply(ctx context.Context, proposalID uuid.UUID) (*ApplyProposalResponse, error) {
	userID := strings.TrimSpace(userIDFromContext(ctx))
	if userID == "" {
		return nil, ErrUnauthorized
	}
	if proposalID == uuid.Nil {
		return nil, ErrInvalidRequest
	}

	proposal, found, err := s.proposalsStorage.Get(ctx, userID, proposalID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrProposalNotFound
	}
	if err := s.ensureProfileOwned(ctx, userID, proposal.ProfileID); err != nil {
		return nil, ErrProposalNotFound
	}
	if proposal.Status != "pending" {
		return nil, ErrNotPending
	}

	switch proposal.Kind {
	case "settings_update":
		patch, err := parseSettingsPatch(proposal.Payload)
		if err != nil {
			return nil, ErrInvalidPayload
		}

		current, err := s.settingsService.GetOrDefault(ctx, userID)
		if err != nil {
			return nil, err
		}
		merged := mergeSettings(current.Settings, patch)
		if err := merged.Validate(); err != nil {
			return nil, ErrInvalidPayload
		}

		updated, err := s.settingsService.Upsert(ctx, userID, merged)
		if err != nil {
			return nil, ErrInvalidPayload
		}

		if err := s.proposalsStorage.UpdateStatus(ctx, userID, proposalID, "applied"); err != nil {
			return nil, err
		}

		return &ApplyProposalResponse{
			Status: "applied",
			Applied: &AppliedResultDTO{
				Settings: &updated,
			},
		}, nil
	case "vitamins_schedule":
		payload, err := parseVitaminsSchedulePayload(proposal.Payload)
		if err != nil {
			return nil, ErrInvalidPayload
		}

		supplementsStorage, ok := s.profileStorage.(storage.SupplementsStorage)
		if !ok || supplementsStorage == nil {
			return nil, ErrUnsupportedKind
		}
		schedulesStorage, ok := s.profileStorage.(storage.SupplementSchedulesStorage)
		if !ok || schedulesStorage == nil {
			return nil, ErrUnsupportedKind
		}

		existingSupplements, err := supplementsStorage.ListSupplements(ctx, proposal.ProfileID)
		if err != nil {
			return nil, err
		}
		byName := make(map[string]storage.Supplement, len(existingSupplements))
		for _, sup := range existingSupplements {
			key := normalizeSupplementName(sup.Name)
			if key == "" {
				continue
			}
			byName[key] = sup
		}

		upserts := make([]storage.ScheduleUpsert, 0, len(payload.Items))
		for _, item := range payload.Items {
			normalizedName := normalizeSupplementName(item.SupplementName)
			supplement, found := byName[normalizedName]
			if !found {
				newSupplement := &storage.Supplement{
					ProfileID: proposal.ProfileID,
					Name:      strings.TrimSpace(item.SupplementName),
				}
				if err := supplementsStorage.CreateSupplement(ctx, newSupplement); err != nil {
					return nil, err
				}
				supplement = *newSupplement
				byName[normalizedName] = supplement
			}

			enabled := true
			if item.IsEnabled != nil {
				enabled = *item.IsEnabled
			}
			upserts = append(upserts, storage.ScheduleUpsert{
				SupplementID: supplement.ID,
				TimeMinutes:  item.TimeMinutes,
				DaysMask:     item.DaysMask,
				IsEnabled:    enabled,
			})
		}

		saved, err := schedulesStorage.ReplaceAll(ctx, userID, proposal.ProfileID, upserts)
		if err != nil {
			return nil, err
		}
		if err := s.proposalsStorage.UpdateStatus(ctx, userID, proposalID, "applied"); err != nil {
			return nil, err
		}
		count := len(saved)
		return &ApplyProposalResponse{
			Status: "applied",
			Applied: &AppliedResultDTO{
				SchedulesCreated: &count,
			},
		}, nil
	case "workout_plan":
		if s.workoutService == nil {
			return nil, ErrUnsupportedKind
		}

		payload, err := parseWorkoutPlanPayload(proposal.Payload)
		if err != nil {
			return nil, ErrInvalidPayload
		}

		// Validate payload
		if !payload.Replace {
			return nil, ErrInvalidPayload
		}
		if len(payload.Items) == 0 || len(payload.Items) > 30 {
			return nil, ErrInvalidPayload
		}

		// Convert to workout service request
		items := make([]workouts.ItemUpsertRequest, 0, len(payload.Items))
		for _, item := range payload.Items {
			items = append(items, workouts.ItemUpsertRequest{
				Kind:        item.Kind,
				TimeMinutes: item.TimeMinutes,
				DaysMask:    item.DaysMask,
				DurationMin: item.DurationMin,
				Intensity:   item.Intensity,
				Note:        item.Note,
				Details:     item.Details,
			})
		}

		req := &workouts.ReplaceItemsRequest{
			ProfileID: proposal.ProfileID,
			Title:     payload.Title,
			Goal:      payload.Goal,
			Replace:   payload.Replace,
			Items:     items,
		}

		_, err = s.workoutService.ReplacePlanAndItems(ctx, req)
		if err != nil {
			return nil, err
		}

		if err := s.proposalsStorage.UpdateStatus(ctx, userID, proposalID, "applied"); err != nil {
			return nil, err
		}

		count := len(payload.Items)
		return &ApplyProposalResponse{
			Status: "applied",
			Applied: &AppliedResultDTO{
				WorkoutItemsCreated: &count,
			},
		}, nil
	case "nutrition_plan":
		if s.nutritionService == nil {
			return nil, ErrUnsupportedKind
		}

		payload, err := parseNutritionPlanPayload(proposal.Payload)
		if err != nil {
			return nil, ErrInvalidPayload
		}

		// Validate payload - all fields are required
		if payload.CaloriesKcal < 800 || payload.CaloriesKcal > 6000 {
			return nil, ErrInvalidPayload
		}
		if payload.ProteinG < 0 || payload.ProteinG > 400 {
			return nil, ErrInvalidPayload
		}
		if payload.FatG < 0 || payload.FatG > 400 {
			return nil, ErrInvalidPayload
		}
		if payload.CarbsG < 0 || payload.CarbsG > 400 {
			return nil, ErrInvalidPayload
		}
		if payload.CalciumMg < 0 || payload.CalciumMg > 5000 {
			return nil, ErrInvalidPayload
		}

		// Call nutrition service to upsert targets
		err = s.nutritionService.UpsertSimple(ctx, userID, proposal.ProfileID,
			payload.CaloriesKcal, payload.ProteinG, payload.FatG, payload.CarbsG, payload.CalciumMg)
		if err != nil {
			return nil, err
		}

		if err := s.proposalsStorage.UpdateStatus(ctx, userID, proposalID, "applied"); err != nil {
			return nil, err
		}

		updated := true
		return &ApplyProposalResponse{
			Status: "applied",
			Applied: &AppliedResultDTO{
				NutritionTargets: &updated,
			},
		}, nil
	case "meal_plan":
		if s.mealPlanService == nil {
			return nil, ErrUnsupportedKind
		}

		payload, err := parseMealPlanPayload(proposal.Payload)
		if err != nil {
			return nil, ErrInvalidPayload
		}

		// Validate payload
		if payload.Title == "" || len(payload.Title) > 200 {
			return nil, ErrInvalidPayload
		}
		if len(payload.Items) == 0 || len(payload.Items) > 28 {
			return nil, ErrInvalidPayload
		}

		// Convert to meal plan service request
		items := make([]mealplans.MealPlanItemUpsertInput, 0, len(payload.Items))
		for _, item := range payload.Items {
			items = append(items, mealplans.MealPlanItemUpsertInput{
				DayIndex:       item.DayIndex,
				MealSlot:       item.MealSlot,
				Title:          item.Title,
				Notes:          item.Notes,
				ApproxKcal:     item.ApproxKcal,
				ApproxProteinG: item.ApproxProteinG,
				ApproxFatG:     item.ApproxFatG,
				ApproxCarbsG:   item.ApproxCarbsG,
			})
		}

		req := mealplans.ReplaceMealPlanRequest{
			ProfileID: proposal.ProfileID.String(),
			Title:     payload.Title,
			Items:     items,
		}

		_, _, err = s.mealPlanService.ReplaceActive(ctx, userID, req)
		if err != nil {
			return nil, err
		}

		if err := s.proposalsStorage.UpdateStatus(ctx, userID, proposalID, "applied"); err != nil {
			return nil, err
		}

		count := len(payload.Items)
		return &ApplyProposalResponse{
			Status: "applied",
			Applied: &AppliedResultDTO{
				MealPlanItemsCreated: &count,
			},
		}, nil
	default:
		return nil, ErrUnsupportedKind
	}
}

func (s *Service) Reject(ctx context.Context, proposalID uuid.UUID) (*RejectProposalResponse, error) {
	userID := strings.TrimSpace(userIDFromContext(ctx))
	if userID == "" {
		return nil, ErrUnauthorized
	}
	if proposalID == uuid.Nil {
		return nil, ErrInvalidRequest
	}

	proposal, found, err := s.proposalsStorage.Get(ctx, userID, proposalID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrProposalNotFound
	}
	if err := s.ensureProfileOwned(ctx, userID, proposal.ProfileID); err != nil {
		return nil, ErrProposalNotFound
	}
	if proposal.Status != "pending" {
		return nil, ErrNotPending
	}

	if err := s.proposalsStorage.UpdateStatus(ctx, userID, proposalID, "rejected"); err != nil {
		return nil, err
	}

	return &RejectProposalResponse{Status: "rejected"}, nil
}

func (s *Service) ensureProfileOwned(ctx context.Context, ownerUserID string, profileID uuid.UUID) error {
	profile, err := s.profileStorage.GetProfile(ctx, profileID)
	if err != nil {
		return ErrProposalNotFound
	}
	if profile.OwnerUserID != ownerUserID {
		return ErrProposalNotFound
	}
	return nil
}

func userIDFromContext(ctx context.Context) string {
	userID, ok := userctx.GetUserID(ctx)
	if !ok {
		return ""
	}
	return strings.TrimSpace(userID)
}

func normalizeStatusFilter(status string) (string, error) {
	switch strings.TrimSpace(status) {
	case "", "pending", "applied", "rejected":
		return strings.TrimSpace(status), nil
	default:
		return "", ErrInvalidRequest
	}
}

type settingsPatch struct {
	TimeZone                  *string `json:"time_zone"`
	QuietStartMinutes         *int    `json:"quiet_start_minutes"`
	QuietEndMinutes           *int    `json:"quiet_end_minutes"`
	NotificationsMaxPerDay    *int    `json:"notifications_max_per_day"`
	MinSleepMinutes           *int    `json:"min_sleep_minutes"`
	MinSteps                  *int    `json:"min_steps"`
	MinActiveEnergyKcal       *int    `json:"min_active_energy_kcal"`
	MorningCheckinTimeMinutes *int    `json:"morning_checkin_time_minutes"`
	EveningCheckinTimeMinutes *int    `json:"evening_checkin_time_minutes"`
	VitaminsTimeMinutes       *int    `json:"vitamins_time_minutes"`
}

type vitaminsSchedulePayload struct {
	Replace bool                          `json:"replace"`
	Items   []vitaminsSchedulePayloadItem `json:"items"`
}

type vitaminsSchedulePayloadItem struct {
	SupplementName string `json:"supplement_name"`
	TimeMinutes    int    `json:"time_minutes"`
	DaysMask       int    `json:"days_mask"`
	IsEnabled      *bool  `json:"is_enabled,omitempty"`
}

func parseSettingsPatch(payload []byte) (settingsPatch, error) {
	var patch settingsPatch

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&patch); err != nil {
		return settingsPatch{}, err
	}

	if patch.TimeZone == nil &&
		patch.QuietStartMinutes == nil &&
		patch.QuietEndMinutes == nil &&
		patch.NotificationsMaxPerDay == nil &&
		patch.MinSleepMinutes == nil &&
		patch.MinSteps == nil &&
		patch.MinActiveEnergyKcal == nil &&
		patch.MorningCheckinTimeMinutes == nil &&
		patch.EveningCheckinTimeMinutes == nil &&
		patch.VitaminsTimeMinutes == nil {
		return settingsPatch{}, ErrInvalidPayload
	}

	return patch, nil
}

func parseVitaminsSchedulePayload(payload []byte) (vitaminsSchedulePayload, error) {
	var parsed vitaminsSchedulePayload
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&parsed); err != nil {
		return vitaminsSchedulePayload{}, err
	}

	if !parsed.Replace {
		return vitaminsSchedulePayload{}, ErrInvalidPayload
	}
	if len(parsed.Items) == 0 || len(parsed.Items) > 20 {
		return vitaminsSchedulePayload{}, ErrInvalidPayload
	}

	cleanItems := make([]vitaminsSchedulePayloadItem, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		name := strings.TrimSpace(item.SupplementName)
		if len(name) < 1 || len(name) > 80 {
			return vitaminsSchedulePayload{}, ErrInvalidPayload
		}
		if item.TimeMinutes < 0 || item.TimeMinutes > 1439 {
			return vitaminsSchedulePayload{}, ErrInvalidPayload
		}
		if item.DaysMask < 0 || item.DaysMask > 127 {
			return vitaminsSchedulePayload{}, ErrInvalidPayload
		}

		enabled := true
		if item.IsEnabled != nil {
			enabled = *item.IsEnabled
		}

		cleanItems = append(cleanItems, vitaminsSchedulePayloadItem{
			SupplementName: name,
			TimeMinutes:    item.TimeMinutes,
			DaysMask:       item.DaysMask,
			IsEnabled:      &enabled,
		})
	}

	parsed.Items = cleanItems
	return parsed, nil
}

func normalizeSupplementName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func mergeSettings(base settings.SettingsDTO, patch settingsPatch) settings.SettingsDTO {
	merged := base

	if patch.TimeZone != nil {
		trimmed := strings.TrimSpace(*patch.TimeZone)
		if trimmed == "" {
			merged.TimeZone = nil
		} else {
			merged.TimeZone = &trimmed
		}
	}
	if patch.QuietStartMinutes != nil {
		v := *patch.QuietStartMinutes
		merged.QuietStartMinutes = &v
	}
	if patch.QuietEndMinutes != nil {
		v := *patch.QuietEndMinutes
		merged.QuietEndMinutes = &v
	}
	if patch.NotificationsMaxPerDay != nil {
		merged.NotificationsMaxPerDay = *patch.NotificationsMaxPerDay
	}
	if patch.MinSleepMinutes != nil {
		merged.MinSleepMinutes = *patch.MinSleepMinutes
	}
	if patch.MinSteps != nil {
		merged.MinSteps = *patch.MinSteps
	}
	if patch.MinActiveEnergyKcal != nil {
		merged.MinActiveEnergyKcal = *patch.MinActiveEnergyKcal
	}
	if patch.MorningCheckinTimeMinutes != nil {
		merged.MorningCheckinTimeMinutes = *patch.MorningCheckinTimeMinutes
	}
	if patch.EveningCheckinTimeMinutes != nil {
		merged.EveningCheckinTimeMinutes = *patch.EveningCheckinTimeMinutes
	}
	if patch.VitaminsTimeMinutes != nil {
		merged.VitaminsTimeMinutes = *patch.VitaminsTimeMinutes
	}

	return merged
}
