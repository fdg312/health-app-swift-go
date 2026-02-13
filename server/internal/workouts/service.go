package workouts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/userctx"
	"github.com/google/uuid"
)

var (
	ErrUnauthorized    = errors.New("unauthorized")
	ErrInvalidRequest  = errors.New("invalid request")
	ErrProfileNotFound = errors.New("profile not found")
	ErrPlanNotFound    = errors.New("plan not found")
	ErrItemNotFound    = errors.New("item not found")
)

// FeedService interface for getting actual workouts from feed/day.
type FeedService interface {
	GetDaySummary(ctx context.Context, profileID uuid.UUID, date string) (interface{}, error)
}

// Service provides workout plan management.
type Service struct {
	plansStorage       storage.WorkoutPlansStorage
	itemsStorage       storage.WorkoutPlanItemsStorage
	completionsStorage storage.WorkoutCompletionsStorage
	profilesStorage    storage.Storage
	feedService        FeedService
}

// NewService creates a new workouts service.
func NewService(
	plansStorage storage.WorkoutPlansStorage,
	itemsStorage storage.WorkoutPlanItemsStorage,
	completionsStorage storage.WorkoutCompletionsStorage,
	profilesStorage storage.Storage,
	feedService FeedService,
) *Service {
	return &Service{
		plansStorage:       plansStorage,
		itemsStorage:       itemsStorage,
		completionsStorage: completionsStorage,
		profilesStorage:    profilesStorage,
		feedService:        feedService,
	}
}

// GetActivePlan returns the active workout plan for a profile.
func (s *Service) GetActivePlan(ctx context.Context, profileID uuid.UUID) (*GetPlanResponse, error) {
	userID := normalizeOwner(userIDFromContext(ctx))
	if userID == "" {
		return nil, ErrUnauthorized
	}
	if profileID == uuid.Nil {
		return nil, ErrInvalidRequest
	}

	// Verify ownership
	if err := s.ensureProfileOwned(ctx, userID, profileID); err != nil {
		return nil, err
	}

	// Get active plan
	plan, found, err := s.plansStorage.GetActivePlan(userID, profileID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &GetPlanResponse{
			Plan:  nil,
			Items: []ItemDTO{},
		}, nil
	}

	// Get items
	items, err := s.itemsStorage.ListItems(userID, profileID, plan.ID)
	if err != nil {
		return nil, err
	}

	planDTO := planToDTO(plan)
	itemDTOs := make([]ItemDTO, 0, len(items))
	for _, item := range items {
		itemDTOs = append(itemDTOs, itemToDTO(item))
	}

	return &GetPlanResponse{
		Plan:  &planDTO,
		Items: itemDTOs,
	}, nil
}

// ReplacePlanAndItems replaces the entire workout plan and all its items.
func (s *Service) ReplacePlanAndItems(ctx context.Context, req *ReplaceItemsRequest) (*ReplaceItemsResponse, error) {
	userID := normalizeOwner(userIDFromContext(ctx))
	if userID == "" {
		return nil, ErrUnauthorized
	}

	// Validate request
	if err := ValidateReplaceItemsRequest(req); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}

	// Verify ownership
	if err := s.ensureProfileOwned(ctx, userID, req.ProfileID); err != nil {
		return nil, err
	}

	// Upsert active plan
	plan, err := s.plansStorage.UpsertActivePlan(userID, req.ProfileID, req.Title, req.Goal)
	if err != nil {
		return nil, err
	}

	// Convert items to storage format
	storageItems := make([]storage.WorkoutItemUpsert, 0, len(req.Items))
	for _, item := range req.Items {
		storageItems = append(storageItems, storage.WorkoutItemUpsert{
			Kind:        item.Kind,
			TimeMinutes: item.TimeMinutes,
			DaysMask:    item.DaysMask,
			DurationMin: item.DurationMin,
			Intensity:   item.Intensity,
			Note:        item.Note,
			Details:     []byte(item.Details),
		})
	}

	// Replace all items
	items, err := s.itemsStorage.ReplaceAllItems(userID, req.ProfileID, plan.ID, storageItems)
	if err != nil {
		return nil, err
	}

	planDTO := planToDTO(plan)
	itemDTOs := make([]ItemDTO, 0, len(items))
	for _, item := range items {
		itemDTOs = append(itemDTOs, itemToDTO(item))
	}

	return &ReplaceItemsResponse{
		Plan:  planDTO,
		Items: itemDTOs,
	}, nil
}

// UpsertCompletion creates or updates a workout completion record.
func (s *Service) UpsertCompletion(ctx context.Context, req *UpsertCompletionRequest) (*CompletionDTO, error) {
	userID := normalizeOwner(userIDFromContext(ctx))
	if userID == "" {
		return nil, ErrUnauthorized
	}

	// Validate request
	if err := ValidateUpsertCompletionRequest(req); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}

	// Verify ownership
	if err := s.ensureProfileOwned(ctx, userID, req.ProfileID); err != nil {
		return nil, err
	}

	// Upsert completion
	completion, err := s.completionsStorage.UpsertCompletion(
		userID,
		req.ProfileID,
		req.Date,
		req.PlanItemID,
		req.Status,
		req.Note,
	)
	if err != nil {
		return nil, err
	}

	dto := completionToDTO(completion)
	return &dto, nil
}

// GetToday returns today's workout plan, completions, and actual workouts.
func (s *Service) GetToday(ctx context.Context, profileID uuid.UUID, date string) (*TodayResponse, error) {
	userID := normalizeOwner(userIDFromContext(ctx))
	if userID == "" {
		return nil, ErrUnauthorized
	}
	if profileID == uuid.Nil {
		return nil, ErrInvalidRequest
	}
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// Validate date format
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid date format", ErrInvalidRequest)
	}

	// Verify ownership
	if err := s.ensureProfileOwned(ctx, userID, profileID); err != nil {
		return nil, err
	}

	// Get active plan
	plan, found, err := s.plansStorage.GetActivePlan(userID, profileID)
	if err != nil {
		return nil, err
	}

	var plannedItems []ItemDTO
	if found {
		// Get all items
		items, err := s.itemsStorage.ListItems(userID, profileID, plan.ID)
		if err != nil {
			return nil, err
		}

		// Filter items for today's weekday
		weekday := int(parsedDate.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday = 7
		}
		weekday-- // 0-indexed (Monday=0, Sunday=6)

		for _, item := range items {
			// Check if this day is in the days_mask
			if item.DaysMask&(1<<weekday) != 0 {
				plannedItems = append(plannedItems, itemToDTO(item))
			}
		}
	}

	if plannedItems == nil {
		plannedItems = []ItemDTO{}
	}

	// Get completions for today
	completions, err := s.completionsStorage.ListCompletions(userID, profileID, date, date)
	if err != nil {
		return nil, err
	}

	completionDTOs := make([]CompletionDTO, 0, len(completions))
	for _, c := range completions {
		completionDTOs = append(completionDTOs, completionToDTO(c))
	}

	// Get actual workouts from feed (if feedService is available)
	var actualWorkouts []WorkoutDTO
	if s.feedService != nil {
		actualWorkouts = s.getActualWorkouts(ctx, profileID, date)
	}
	if actualWorkouts == nil {
		actualWorkouts = []WorkoutDTO{}
	}

	// Calculate isDone
	isDone := s.calculateIsDone(plannedItems, completionDTOs, actualWorkouts)

	return &TodayResponse{
		Date:           date,
		ProfileID:      profileID,
		Planned:        plannedItems,
		Completions:    completionDTOs,
		ActualWorkouts: actualWorkouts,
		IsDone:         isDone,
	}, nil
}

// ListCompletions returns completions in a date range.
func (s *Service) ListCompletions(ctx context.Context, profileID uuid.UUID, from, to string) (*ListCompletionsResponse, error) {
	userID := normalizeOwner(userIDFromContext(ctx))
	if userID == "" {
		return nil, ErrUnauthorized
	}
	if profileID == uuid.Nil {
		return nil, ErrInvalidRequest
	}

	// Verify ownership
	if err := s.ensureProfileOwned(ctx, userID, profileID); err != nil {
		return nil, err
	}

	// Validate date formats
	if _, err := time.Parse("2006-01-02", from); err != nil {
		return nil, fmt.Errorf("%w: invalid from date", ErrInvalidRequest)
	}
	if _, err := time.Parse("2006-01-02", to); err != nil {
		return nil, fmt.Errorf("%w: invalid to date", ErrInvalidRequest)
	}

	completions, err := s.completionsStorage.ListCompletions(userID, profileID, from, to)
	if err != nil {
		return nil, err
	}

	dtos := make([]CompletionDTO, 0, len(completions))
	for _, c := range completions {
		dtos = append(dtos, completionToDTO(c))
	}

	return &ListCompletionsResponse{
		Completions: dtos,
	}, nil
}

// ============================================================================
// Helper methods
// ============================================================================

func (s *Service) ensureProfileOwned(ctx context.Context, userID string, profileID uuid.UUID) error {
	profile, err := s.profilesStorage.GetProfile(ctx, profileID)
	if err != nil {
		return ErrProfileNotFound
	}
	if normalizeOwner(profile.OwnerUserID) != userID {
		return ErrProfileNotFound // Don't reveal existence
	}
	return nil
}

func (s *Service) getActualWorkouts(ctx context.Context, profileID uuid.UUID, date string) []WorkoutDTO {
	if s.feedService == nil {
		return []WorkoutDTO{}
	}

	summary, err := s.feedService.GetDaySummary(ctx, profileID, date)
	if err != nil {
		return []WorkoutDTO{}
	}

	// Try to extract workouts from summary (interface{})
	var workouts []WorkoutDTO
	if summaryMap, ok := summary.(map[string]interface{}); ok {
		if sessions, ok := summaryMap["sessions"].(map[string]interface{}); ok {
			if workoutsRaw, ok := sessions["workouts"].([]interface{}); ok {
				for _, w := range workoutsRaw {
					if workout, ok := w.(map[string]interface{}); ok {
						dto := WorkoutDTO{}
						if start, ok := workout["start"].(string); ok {
							if t, err := time.Parse(time.RFC3339, start); err == nil {
								dto.Start = t
							}
						}
						if end, ok := workout["end"].(string); ok {
							if t, err := time.Parse(time.RFC3339, end); err == nil {
								dto.End = t
							}
						}
						if label, ok := workout["label"].(string); ok {
							dto.Label = label
						}
						if calories, ok := workout["calories_kcal"].(float64); ok {
							cal := int(calories)
							dto.CaloriesKcal = &cal
						}
						workouts = append(workouts, dto)
					}
				}
			}
		}
	}

	return workouts
}

func (s *Service) calculateIsDone(planned []ItemDTO, completions []CompletionDTO, actualWorkouts []WorkoutDTO) bool {
	if len(planned) == 0 {
		// No planned workouts: done if there are actual workouts
		return len(actualWorkouts) > 0
	}

	// Create completion map
	completionMap := make(map[uuid.UUID]bool)
	for _, c := range completions {
		if c.Status == "done" || c.Status == "skipped" {
			completionMap[c.PlanItemID] = true
		}
	}

	// Check if all planned items have completions
	allCompleted := true
	for _, item := range planned {
		if !completionMap[item.ID] {
			allCompleted = false
			break
		}
	}

	if allCompleted {
		return true
	}

	// Alternative: if there are actual workouts and at least one planned item is done
	if len(actualWorkouts) > 0 && len(completions) > 0 {
		return true
	}

	return false
}

// ============================================================================
// Converters
// ============================================================================

func planToDTO(plan storage.WorkoutPlan) PlanDTO {
	return PlanDTO{
		ID:        plan.ID,
		ProfileID: plan.ProfileID,
		Title:     plan.Title,
		Goal:      plan.Goal,
		IsActive:  plan.IsActive,
		CreatedAt: plan.CreatedAt,
		UpdatedAt: plan.UpdatedAt,
	}
}

func itemToDTO(item storage.WorkoutPlanItem) ItemDTO {
	details := json.RawMessage(item.Details)
	if len(details) == 0 {
		details = json.RawMessage("{}")
	}
	return ItemDTO{
		ID:          item.ID,
		Kind:        item.Kind,
		TimeMinutes: item.TimeMinutes,
		DaysMask:    item.DaysMask,
		DurationMin: item.DurationMin,
		Intensity:   item.Intensity,
		Note:        item.Note,
		Details:     details,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func completionToDTO(completion storage.WorkoutCompletion) CompletionDTO {
	return CompletionDTO{
		ID:         completion.ID,
		Date:       completion.Date,
		PlanItemID: completion.PlanItemID,
		Status:     completion.Status,
		Note:       completion.Note,
		CreatedAt:  completion.CreatedAt,
		UpdatedAt:  completion.UpdatedAt,
	}
}

func userIDFromContext(ctx context.Context) string {
	userID, _ := userctx.GetUserID(ctx)
	return userID
}

func normalizeOwner(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}
