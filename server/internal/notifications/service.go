package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fdg312/health-hub/internal/checkins"
	"github.com/fdg312/health-hub/internal/config"
	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/userctx"
	"github.com/google/uuid"
)

// WorkoutPlansStorage interface for checking planned workouts
type WorkoutPlansStorage interface {
	GetActivePlan(ownerUserID string, profileID uuid.UUID) (storage.WorkoutPlan, bool, error)
}

// WorkoutPlanItemsStorage interface for getting workout items
type WorkoutPlanItemsStorage interface {
	ListItems(ownerUserID string, profileID uuid.UUID, planID uuid.UUID) ([]storage.WorkoutPlanItem, error)
}

// WorkoutCompletionsStorage interface for checking completions
type WorkoutCompletionsStorage interface {
	ListCompletions(ownerUserID string, profileID uuid.UUID, from string, to string) ([]storage.WorkoutCompletion, error)
}

// MealPlansStorage interface for checking meal plans
type MealPlansStorage interface {
	GetToday(ctx context.Context, ownerUserID string, profileID string, date time.Time) ([]storage.MealPlanItem, error)
	GetActive(ctx context.Context, ownerUserID string, profileID string) (storage.MealPlan, []storage.MealPlanItem, bool, error)
}

type Service struct {
	storage            storage.NotificationsStorage
	metrics            storage.MetricsStorage
	checkins           checkins.Storage
	profiles           storage.Storage
	settings           storage.SettingsStorage
	config             *config.Config
	workoutPlans       WorkoutPlansStorage
	workoutItems       WorkoutPlanItemsStorage
	workoutCompletions WorkoutCompletionsStorage
	mealPlans          MealPlansStorage
}

func NewService(storage storage.NotificationsStorage, metrics storage.MetricsStorage, checkins checkins.Storage, profiles storage.Storage, settings storage.SettingsStorage, cfg *config.Config) *Service {
	return &Service{
		storage:  storage,
		metrics:  metrics,
		checkins: checkins,
		profiles: profiles,
		settings: settings,
		config:   cfg,
	}
}

// WithWorkoutStorages adds workout storage interfaces for workout reminders
func (s *Service) WithWorkoutStorages(plans WorkoutPlansStorage, items WorkoutPlanItemsStorage, completions WorkoutCompletionsStorage) *Service {
	s.workoutPlans = plans
	s.workoutItems = items
	s.workoutCompletions = completions
	return s
}

// WithMealPlansStorage adds meal plans storage for meal plan reminders
func (s *Service) WithMealPlansStorage(mealPlans MealPlansStorage) *Service {
	s.mealPlans = mealPlans
	return s
}

func (s *Service) ListNotifications(ctx context.Context, profileID uuid.UUID, onlyUnread bool, limit, offset int) ([]NotificationDTO, error) {
	if err := s.ensureProfileAccess(ctx, profileID); err != nil {
		return nil, err
	}

	notifications, err := s.storage.ListNotifications(ctx, profileID, onlyUnread, limit, offset)
	if err != nil {
		return nil, err
	}

	dtos := make([]NotificationDTO, len(notifications))
	for i, n := range notifications {
		dtos[i] = toDTO(&n)
	}

	return dtos, nil
}

func (s *Service) UnreadCount(ctx context.Context, profileID uuid.UUID) (int, error) {
	if err := s.ensureProfileAccess(ctx, profileID); err != nil {
		return 0, err
	}
	return s.storage.UnreadCount(ctx, profileID)
}

func (s *Service) MarkRead(ctx context.Context, profileID uuid.UUID, ids []uuid.UUID) (int, error) {
	if err := s.ensureProfileAccess(ctx, profileID); err != nil {
		return 0, err
	}
	return s.storage.MarkRead(ctx, profileID, ids)
}

func (s *Service) MarkAllRead(ctx context.Context, profileID uuid.UUID) (int, error) {
	if err := s.ensureProfileAccess(ctx, profileID); err != nil {
		return 0, err
	}
	return s.storage.MarkAllRead(ctx, profileID)
}

// Generate creates notifications based on metrics and checkins for a given date
func (s *Service) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	profile, err := s.getProfileIfAuthorized(ctx, req.ProfileID)
	if err != nil {
		return nil, err
	}

	// Parse date
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	effective, err := s.loadEffectiveSettings(ctx, profile.OwnerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user settings: %w", err)
	}

	// Load timezone
	loc, err := time.LoadLocation(effective.TimeZone)
	if err != nil {
		loc = time.UTC
	}

	// Get daily metrics for this date
	metrics, err := s.metrics.GetDailyMetrics(ctx, req.ProfileID, req.Date, req.Date)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	// Get checkins for this date
	checkinsList, err := s.checkins.ListCheckins(req.ProfileID, req.Date, req.Date)
	if err != nil {
		return nil, fmt.Errorf("failed to get checkins: %w", err)
	}

	// Build checkin map
	checkinsMap := make(map[string]bool)
	for _, c := range checkinsList {
		checkinsMap[c.Type] = true
	}

	// Parse metrics JSONB if available
	var dailyData map[string]interface{}
	if len(metrics) > 0 {
		if err := json.Unmarshal(metrics[0].Payload, &dailyData); err == nil {
			// parsed successfully
		}
	}

	// Generate notifications
	var candidates []storage.Notification
	sourceDate := date

	// 1. Low sleep
	if sleep, ok := dailyData["sleep"].(map[string]interface{}); ok {
		if totalMin, ok := sleep["total_minutes"].(float64); ok {
			if int(totalMin) < effective.MinSleepMinutes {
				hours := int(totalMin) / 60
				minutes := int(totalMin) % 60
				thresholdHours := effective.MinSleepMinutes / 60
				candidates = append(candidates, storage.Notification{
					ProfileID:  req.ProfileID,
					Kind:       "low_sleep",
					Title:      "Плохой сон",
					Body:       fmt.Sprintf("Сон: %dч %dм (ниже %dч). Попробуй лечь раньше сегодня.", hours, minutes, thresholdHours),
					SourceDate: &sourceDate,
					Severity:   "warn",
				})
			}
		}
	}

	// 2. Low activity
	if activity, ok := dailyData["activity"].(map[string]interface{}); ok {
		steps := 0
		if s, ok := activity["steps"].(float64); ok {
			steps = int(s)
		}
		activeEnergy := 0
		if ae, ok := activity["active_energy_kcal"].(float64); ok {
			activeEnergy = int(ae)
		}

		if (steps > 0 && steps < effective.MinSteps) || (activeEnergy > 0 && activeEnergy < effective.MinActiveEnergyKcal) {
			severity := "info"
			if steps < effective.MinSteps/2 {
				severity = "warn"
			}
			candidates = append(candidates, storage.Notification{
				ProfileID:  req.ProfileID,
				Kind:       "low_activity",
				Title:      "Низкая активность",
				Body:       fmt.Sprintf("Шаги: %d. Небольшая прогулка 10–15 минут поможет.", steps),
				SourceDate: &sourceDate,
				Severity:   severity,
			})
		}
	}

	// 3. Missing morning checkin (only if date == today and time > 12:00)
	if isToday(date, req.Now, loc) && !checkinsMap["morning"] {
		if minutesOfDay(req.Now.In(loc)) >= effective.MorningCheckinTimeMinutes {
			candidates = append(candidates, storage.Notification{
				ProfileID:  req.ProfileID,
				Kind:       "missing_morning_checkin",
				Title:      "Пропущен утренний чек-ин",
				Body:       "Как прошло утро? Заполни утренний чек-ин.",
				SourceDate: &sourceDate,
				Severity:   "info",
			})
		}
	}

	// 4. Missing evening checkin
	if isToday(date, req.Now, loc) && !checkinsMap["evening"] {
		if minutesOfDay(req.Now.In(loc)) >= effective.EveningCheckinTimeMinutes {
			candidates = append(candidates, storage.Notification{
				ProfileID:  req.ProfileID,
				Kind:       "missing_evening_checkin",
				Title:      "Пропущен вечерний чек-ин",
				Body:       "Как прошёл день? Заполни вечерний чек-ин.",
				SourceDate: &sourceDate,
				Severity:   "info",
			})
		}
	}

	// 5. Vitamins reminder based on supplement schedules.
	vitaminsReminder, err := s.maybeBuildVitaminsReminder(ctx, profile, req, effective, loc)
	if err != nil {
		return nil, fmt.Errorf("failed to build vitamins reminder: %w", err)
	}
	if vitaminsReminder != nil {
		candidates = append(candidates, *vitaminsReminder)
	}

	// 6. Workout reminder based on workout plan
	workoutReminder, err := s.maybeBuildWorkoutReminder(ctx, profile, req, effective, loc)
	if err != nil {
		return nil, fmt.Errorf("failed to build workout reminder: %w", err)
	}
	if workoutReminder != nil {
		candidates = append(candidates, *workoutReminder)
	}

	// 7. Meal plan reminder
	mealPlanReminder, err := s.maybeBuildMealPlanReminder(ctx, profile, req, effective, loc)
	if err != nil {
		return nil, fmt.Errorf("failed to build meal plan reminder: %w", err)
	}
	if mealPlanReminder != nil {
		candidates = append(candidates, *mealPlanReminder)
	}

	// Quiet hours: suppress info reminders, keep warn notifications.
	if effective.QuietEnabled && isInQuietHours(minutesOfDay(req.Now.In(loc)), effective.QuietStartMinutes, effective.QuietEndMinutes) {
		candidates = filterBySeverity(candidates, "warn")
	}

	// Apply max per day limit: prioritize warn > info, and limit
	maxPerDay := effective.NotificationsMaxPerDay
	candidates = applyPriorityAndLimit(candidates, maxPerDay)

	// Create notifications (upsert)
	created := 0
	updated := 0
	for _, candidate := range candidates {
		// Try to create (will upsert if exists)
		err := s.storage.CreateNotification(ctx, &candidate)
		if err != nil {
			// If upsert, treat as update
			// Memory storage doesn't return distinct error, assume creation
			created++
		} else {
			created++
		}
	}

	skipped := len(candidates) - created - updated

	return &GenerateResponse{
		Created: created,
		Updated: updated,
		Skipped: skipped,
	}, nil
}

// Helper: check if date is today in given timezone
func isToday(date time.Time, now time.Time, loc *time.Location) bool {
	nowLocal := now.In(loc)
	y1, m1, d1 := date.Date()
	y2, m2, d2 := nowLocal.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// applyPriorityAndLimit sorts candidates by severity (warn first) and limits count
func applyPriorityAndLimit(candidates []storage.Notification, maxCount int) []storage.Notification {
	if len(candidates) <= maxCount {
		return candidates
	}

	// Sort: warn before info
	sorted := make([]storage.Notification, len(candidates))
	copy(sorted, candidates)

	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Severity == "info" && sorted[j].Severity == "warn" {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted[:maxCount]
}

func toDTO(n *storage.Notification) NotificationDTO {
	dto := NotificationDTO{
		ID:        n.ID,
		ProfileID: n.ProfileID,
		Kind:      n.Kind,
		Title:     n.Title,
		Body:      n.Body,
		Severity:  n.Severity,
		CreatedAt: n.CreatedAt,
		ReadAt:    n.ReadAt,
	}

	if n.SourceDate != nil {
		dateStr := n.SourceDate.Format("2006-01-02")
		dto.SourceDate = &dateStr
	}

	return dto
}

func (s *Service) ensureProfileAccess(ctx context.Context, profileID uuid.UUID) error {
	_, err := s.getProfileIfAuthorized(ctx, profileID)
	return err
}

// maybeBuildWorkoutReminder generates a workout reminder if there are planned workouts today
// that haven't been completed yet.
func (s *Service) maybeBuildWorkoutReminder(ctx context.Context, profile *storage.Profile, req *GenerateRequest, effective effectiveSettings, loc *time.Location) (*storage.Notification, error) {
	// Skip if workout storages not configured
	if s.workoutPlans == nil || s.workoutItems == nil || s.workoutCompletions == nil {
		return nil, nil
	}

	// Parse date
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, err
	}

	// Only generate for today
	if !isToday(date, req.Now, loc) {
		return nil, nil
	}

	// Get active plan
	plan, found, err := s.workoutPlans.GetActivePlan(profile.OwnerUserID, req.ProfileID)
	if err != nil || !found {
		return nil, err
	}

	// Get all items
	items, err := s.workoutItems.ListItems(profile.OwnerUserID, req.ProfileID, plan.ID)
	if err != nil {
		return nil, err
	}

	// Filter items for today's weekday
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7
	}
	weekday-- // 0-indexed (Monday=0, Sunday=6)

	var plannedToday []storage.WorkoutPlanItem
	for _, item := range items {
		// Check if this day is in the days_mask
		if item.DaysMask&(1<<weekday) != 0 {
			plannedToday = append(plannedToday, item)
		}
	}

	// No planned workouts today
	if len(plannedToday) == 0 {
		return nil, nil
	}

	// Check if already completed
	completions, err := s.workoutCompletions.ListCompletions(profile.OwnerUserID, req.ProfileID, req.Date, req.Date)
	if err != nil {
		return nil, err
	}

	// Build completion map
	completedItemIDs := make(map[uuid.UUID]bool)
	for _, comp := range completions {
		if comp.Status == "done" || comp.Status == "skipped" {
			completedItemIDs[comp.PlanItemID] = true
		}
	}

	// Check if all planned items are completed
	allCompleted := true
	for _, item := range plannedToday {
		if !completedItemIDs[item.ID] {
			allCompleted = false
			break
		}
	}

	// If all completed, no reminder needed
	if allCompleted {
		return nil, nil
	}

	// Check if it's close to workout time (30 minutes before earliest planned time)
	currentMinutes := minutesOfDay(req.Now.In(loc))
	minTime := 1440 // max minutes in day
	for _, item := range plannedToday {
		if item.TimeMinutes < minTime {
			minTime = item.TimeMinutes
		}
	}

	// Only remind if current time is within 30 minutes before workout or after
	if minTime != 1440 && currentMinutes >= minTime-30 {
		// Build notification text
		var body string
		if len(plannedToday) == 1 {
			kindRus := translateWorkoutKind(plannedToday[0].Kind)
			body = fmt.Sprintf("Сегодня тренировка: %s • %d мин", kindRus, plannedToday[0].DurationMin)
		} else {
			body = fmt.Sprintf("Сегодня тренировка: %d запланировано", len(plannedToday))
		}

		sourceDate := date
		return &storage.Notification{
			ProfileID:  req.ProfileID,
			Kind:       "workout_reminder",
			Title:      "Тренировка сегодня",
			Body:       body,
			SourceDate: &sourceDate,
			Severity:   "info",
		}, nil
	}

	return nil, nil
}

// translateWorkoutKind translates workout kind to Russian
func translateWorkoutKind(kind string) string {
	switch kind {
	case "run":
		return "бег"
	case "walk":
		return "прогулка"
	case "strength":
		return "силовая"
	case "morning":
		return "утренняя зарядка"
	case "core":
		return "кор"
	case "other":
		return "другое"
	default:
		return kind
	}
}

// maybeBuildMealPlanReminder generates a meal plan reminder if there's an active meal plan
// with meals scheduled for today.
func (s *Service) maybeBuildMealPlanReminder(ctx context.Context, profile *storage.Profile, req *GenerateRequest, effective effectiveSettings, loc *time.Location) (*storage.Notification, error) {
	// Skip if meal plans storage not configured
	if s.mealPlans == nil {
		return nil, nil
	}

	// Parse date
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, err
	}

	// Only generate for today
	if !isToday(date, req.Now, loc) {
		return nil, nil
	}

	// Get active plan
	_, _, found, err := s.mealPlans.GetActive(ctx, profile.OwnerUserID, req.ProfileID.String())
	if err != nil || !found {
		return nil, err
	}

	// Get meals for today
	items, err := s.mealPlans.GetToday(ctx, profile.OwnerUserID, req.ProfileID.String(), date)
	if err != nil || len(items) == 0 {
		return nil, err
	}

	// Generate reminder once per day in the morning (after 8:00)
	currentMinutes := minutesOfDay(req.Now.In(loc))
	if currentMinutes < 480 { // 8:00 AM
		return nil, nil
	}

	// Build notification text
	var body string
	if len(items) == 1 {
		body = fmt.Sprintf("План питания на сегодня: %s", items[0].Title)
	} else {
		body = fmt.Sprintf("План питания на сегодня: %d приёмов пищи", len(items))
	}

	sourceDate := date
	return &storage.Notification{
		ProfileID:  req.ProfileID,
		Kind:       "meal_plan_reminder",
		Title:      "План питания",
		Body:       body,
		SourceDate: &sourceDate,
		Severity:   "info",
	}, nil
}

func (s *Service) getProfileIfAuthorized(ctx context.Context, profileID uuid.UUID) (*storage.Profile, error) {
	profile, err := s.profiles.GetProfile(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("profile_not_found")
	}

	if userID, ok := userctx.GetUserID(ctx); ok && strings.TrimSpace(userID) != "" && profile.OwnerUserID != userID {
		return nil, fmt.Errorf("profile_not_found")
	}

	return profile, nil
}

type effectiveSettings struct {
	TimeZone string

	QuietEnabled      bool
	QuietStartMinutes int
	QuietEndMinutes   int

	NotificationsMaxPerDay    int
	MinSleepMinutes           int
	MinSteps                  int
	MinActiveEnergyKcal       int
	MorningCheckinTimeMinutes int
	EveningCheckinTimeMinutes int
	VitaminsTimeMinutes       int
}

func (s *Service) loadEffectiveSettings(ctx context.Context, ownerUserID string) (effectiveSettings, error) {
	effective := effectiveSettings{
		TimeZone:                  "UTC",
		NotificationsMaxPerDay:    s.config.NotificationsMaxPerDay,
		MinSleepMinutes:           s.config.DefaultSleepMinMinutes,
		MinSteps:                  s.config.DefaultStepsMin,
		MinActiveEnergyKcal:       s.config.DefaultActiveEnergyMinKcal,
		MorningCheckinTimeMinutes: 540,
		EveningCheckinTimeMinutes: 1260,
		VitaminsTimeMinutes:       720,
	}

	if s.settings == nil {
		return effective, nil
	}

	row, found, err := s.settings.GetSettings(ctx, ownerUserID)
	if err != nil {
		return effective, err
	}
	if !found {
		return effective, nil
	}

	if row.TimeZone != nil && strings.TrimSpace(*row.TimeZone) != "" {
		effective.TimeZone = strings.TrimSpace(*row.TimeZone)
	}
	if row.QuietStartMinutes != nil && row.QuietEndMinutes != nil {
		effective.QuietEnabled = true
		effective.QuietStartMinutes = *row.QuietStartMinutes
		effective.QuietEndMinutes = *row.QuietEndMinutes
	}

	effective.NotificationsMaxPerDay = row.NotificationsMaxPerDay
	effective.MinSleepMinutes = row.MinSleepMinutes
	effective.MinSteps = row.MinSteps
	effective.MinActiveEnergyKcal = row.MinActiveEnergyKcal
	effective.MorningCheckinTimeMinutes = row.MorningCheckinMinute
	effective.EveningCheckinTimeMinutes = row.EveningCheckinMinute
	effective.VitaminsTimeMinutes = row.VitaminsTimeMinute

	return effective, nil
}

func (s *Service) maybeBuildVitaminsReminder(
	ctx context.Context,
	profile *storage.Profile,
	req *GenerateRequest,
	effective effectiveSettings,
	loc *time.Location,
) (*storage.Notification, error) {
	schedulesStorage, ok := s.profiles.(storage.SupplementSchedulesStorage)
	if !ok || schedulesStorage == nil {
		return nil, nil
	}
	supplementsStorage, ok := s.profiles.(storage.SupplementsStorage)
	if !ok || supplementsStorage == nil {
		return nil, nil
	}
	intakesStorage, ok := s.profiles.(storage.IntakesStorage)
	if !ok || intakesStorage == nil {
		return nil, nil
	}

	schedules, err := schedulesStorage.ListSchedules(ctx, profile.OwnerUserID, profile.ID)
	if err != nil {
		return nil, err
	}
	if len(schedules) == 0 {
		return nil, nil
	}

	// Only for today's date in user's timezone.
	sourceDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, nil
	}
	if !isToday(sourceDate, req.Now, loc) {
		return nil, nil
	}

	nowLocal := req.Now.In(loc)
	nowMinutes := minutesOfDay(nowLocal)
	weekdayBit := weekdayMaskBit(nowLocal.Weekday())

	activeToday := make([]storage.SupplementSchedule, 0, len(schedules))
	nearestMinutes := -1
	nearestDistance := 24 * 60
	for _, row := range schedules {
		if !row.IsEnabled {
			continue
		}
		if !isWeekdayEnabled(row.DaysMask, weekdayBit) {
			continue
		}
		activeToday = append(activeToday, row)

		d := absMinutes(row.TimeMinutes - nowMinutes)
		if d < nearestDistance {
			nearestDistance = d
			nearestMinutes = row.TimeMinutes
		}
	}
	if len(activeToday) == 0 {
		return nil, nil
	}

	triggerAt := effective.VitaminsTimeMinutes
	if nearestMinutes >= 0 {
		triggerAt = nearestMinutes
	}
	if nowMinutes < triggerAt {
		return nil, nil
	}

	supplements, err := supplementsStorage.ListSupplements(ctx, profile.ID)
	if err != nil {
		return nil, err
	}
	supplementsByID := make(map[uuid.UUID]string, len(supplements))
	for _, sup := range supplements {
		supplementsByID[sup.ID] = strings.TrimSpace(sup.Name)
	}

	dailyStatuses, err := intakesStorage.GetSupplementDaily(ctx, profile.ID, req.Date)
	if err != nil {
		return nil, err
	}

	pendingNamesSet := make(map[string]struct{})
	for _, row := range activeToday {
		if dailyStatuses[row.SupplementID] == "taken" {
			continue
		}
		name := strings.TrimSpace(supplementsByID[row.SupplementID])
		if name == "" {
			continue
		}
		pendingNamesSet[name] = struct{}{}
	}
	if len(pendingNamesSet) == 0 {
		return nil, nil
	}

	pendingNames := make([]string, 0, len(pendingNamesSet))
	for name := range pendingNamesSet {
		pendingNames = append(pendingNames, name)
	}
	sort.Strings(pendingNames)

	n := storage.Notification{
		ProfileID:  profile.ID,
		Kind:       "vitamins_reminder",
		Title:      "Напоминание о витаминах",
		Body:       "Не забудьте принять витамины: " + strings.Join(pendingNames, ", "),
		SourceDate: &sourceDate,
		Severity:   "info",
	}
	return &n, nil
}

func filterBySeverity(candidates []storage.Notification, keepSeverity string) []storage.Notification {
	if len(candidates) == 0 {
		return candidates
	}

	filtered := make([]storage.Notification, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Severity == keepSeverity {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func minutesOfDay(t time.Time) int {
	return t.Hour()*60 + t.Minute()
}

func isInQuietHours(current, start, end int) bool {
	if start == end {
		return true
	}
	if start < end {
		return current >= start && current < end
	}
	return current >= start || current < end
}

func weekdayMaskBit(wd time.Weekday) int {
	if wd == time.Sunday {
		return 6
	}
	return int(wd) - 1 // Monday=1 -> bit 0
}

func isWeekdayEnabled(daysMask, bit int) bool {
	if bit < 0 || bit > 6 {
		return false
	}
	return (daysMask & (1 << bit)) != 0
}

func absMinutes(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
