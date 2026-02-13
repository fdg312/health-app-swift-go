package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

var (
	ErrNotFound = errors.New("profile not found")
)

// MemoryStorage — in-memory реализация Storage и MetricsStorage
type MemoryStorage struct {
	mu                 sync.RWMutex
	profiles           map[uuid.UUID]storage.Profile
	metrics            *MetricsMemoryStorage
	checkins           *CheckinsMemoryStorage
	reports            *ReportsMemoryStorage
	sources            *SourcesMemoryStorage
	notifications      *NotificationsMemoryStorage
	supplements        *SupplementsMemoryStorage
	schedules          *SupplementSchedulesMemoryStorage
	intakes            *IntakesMemoryStorage
	emailOTPs          *EmailOTPMemoryStorage
	settings           *SettingsMemoryStorage
	chat               *ChatMemoryStorage
	proposals          *ProposalsMemoryStorage
	workoutPlans       *WorkoutPlansStorage
	workoutItems       *WorkoutPlanItemsStorage
	workoutCompletions *WorkoutCompletionsStorage
	nutritionTargets   *nutritionTargetsStorage
	foodPrefs          *foodPrefsStorage
	mealPlans          *mealPlansStorage
}

// New создаёт новый MemoryStorage с owner профилем по умолчанию
func New() *MemoryStorage {
	ownerID := uuid.New()
	owner := storage.Profile{
		ID:          ownerID,
		OwnerUserID: "default",
		Type:        "owner",
		Name:        "Я",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return &MemoryStorage{
		profiles: map[uuid.UUID]storage.Profile{
			ownerID: owner,
		},
		metrics:            NewMetricsStorage(),
		checkins:           NewCheckinsMemoryStorage(),
		reports:            NewReportsMemoryStorage(),
		sources:            NewSourcesMemoryStorage(),
		notifications:      NewNotificationsMemoryStorage(),
		supplements:        NewSupplementsMemoryStorage(),
		schedules:          NewSupplementSchedulesMemoryStorage(),
		intakes:            NewIntakesMemoryStorage(),
		emailOTPs:          NewEmailOTPMemoryStorage(),
		settings:           NewSettingsMemoryStorage(),
		chat:               NewChatMemoryStorage(),
		proposals:          NewProposalsMemoryStorage(),
		workoutPlans:       NewWorkoutPlansStorage(),
		workoutItems:       NewWorkoutPlanItemsStorage(),
		workoutCompletions: NewWorkoutCompletionsStorage(),
		nutritionTargets:   newNutritionTargetsStorage(),
		foodPrefs:          newFoodPrefsStorage(),
		mealPlans:          newMealPlansStorage(),
	}
}

func (m *MemoryStorage) ListProfiles(ctx context.Context) ([]storage.Profile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	profiles := make([]storage.Profile, 0, len(m.profiles))
	for _, p := range m.profiles {
		profiles = append(profiles, p)
	}

	return profiles, nil
}

func (m *MemoryStorage) GetProfile(ctx context.Context, id uuid.UUID) (*storage.Profile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.profiles[id]
	if !ok {
		return nil, ErrNotFound
	}

	return &p, nil
}

func (m *MemoryStorage) CreateProfile(ctx context.Context, profile *storage.Profile) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}

	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()

	m.profiles[profile.ID] = *profile

	return nil
}

func (m *MemoryStorage) UpdateProfile(ctx context.Context, profile *storage.Profile) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.profiles[profile.ID]; !ok {
		return ErrNotFound
	}

	profile.UpdatedAt = time.Now()
	m.profiles[profile.ID] = *profile

	return nil
}

func (m *MemoryStorage) DeleteProfile(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.profiles[id]; !ok {
		return ErrNotFound
	}

	delete(m.profiles, id)

	return nil
}

func (m *MemoryStorage) Close() error {
	// no-op для memory
	return nil
}

// MetricsStorage methods - делегируем к встроенному metrics storage

func (m *MemoryStorage) UpsertDailyMetric(ctx context.Context, profileID uuid.UUID, date string, payload []byte) error {
	return m.metrics.UpsertDailyMetric(ctx, profileID, date, payload)
}

func (m *MemoryStorage) GetDailyMetrics(ctx context.Context, profileID uuid.UUID, from, to string) ([]storage.DailyMetricRow, error) {
	return m.metrics.GetDailyMetrics(ctx, profileID, from, to)
}

func (m *MemoryStorage) UpsertHourlyMetric(ctx context.Context, profileID uuid.UUID, hour time.Time, steps *int, hrMin, hrMax, hrAvg *int) error {
	return m.metrics.UpsertHourlyMetric(ctx, profileID, hour, steps, hrMin, hrMax, hrAvg)
}

func (m *MemoryStorage) GetHourlyMetrics(ctx context.Context, profileID uuid.UUID, date string) ([]storage.HourlyMetricRow, error) {
	return m.metrics.GetHourlyMetrics(ctx, profileID, date)
}

func (m *MemoryStorage) InsertSleepSegment(ctx context.Context, profileID uuid.UUID, start, end time.Time, stage string) error {
	return m.metrics.InsertSleepSegment(ctx, profileID, start, end, stage)
}

func (m *MemoryStorage) InsertWorkout(ctx context.Context, profileID uuid.UUID, start, end time.Time, label string, caloriesKcal *int) error {
	return m.metrics.InsertWorkout(ctx, profileID, start, end, label, caloriesKcal)
}

// GetCheckinsStorage returns the checkins storage
func (m *MemoryStorage) GetCheckinsStorage() *CheckinsMemoryStorage {
	return m.checkins
}

// GetReportsStorage returns the reports storage
func (m *MemoryStorage) GetReportsStorage() *ReportsMemoryStorage {
	return m.reports
}

// GetSourcesStorage returns the sources storage
func (m *MemoryStorage) GetSourcesStorage() *SourcesMemoryStorage {
	return m.sources
}

// SourcesStorage methods - delegate to embedded sources storage

func (m *MemoryStorage) CreateSource(ctx context.Context, source *storage.Source) error {
	return m.sources.CreateSource(ctx, source)
}

func (m *MemoryStorage) GetSource(ctx context.Context, id uuid.UUID) (*storage.Source, error) {
	return m.sources.GetSource(ctx, id)
}

func (m *MemoryStorage) ListSources(ctx context.Context, profileID uuid.UUID, query string, checkinID *uuid.UUID, limit, offset int) ([]storage.Source, error) {
	return m.sources.ListSources(ctx, profileID, query, checkinID, limit, offset)
}

func (m *MemoryStorage) DeleteSource(ctx context.Context, id uuid.UUID) error {
	return m.sources.DeleteSource(ctx, id)
}

func (m *MemoryStorage) GetSourceBlob(ctx context.Context, sourceID uuid.UUID) ([]byte, string, error) {
	return m.sources.GetSourceBlob(ctx, sourceID)
}

func (m *MemoryStorage) PutSourceBlob(ctx context.Context, sourceID uuid.UUID, data []byte, contentType string) error {
	return m.sources.PutSourceBlob(ctx, sourceID, data, contentType)
}

// GetNotificationsStorage returns the notifications storage
func (m *MemoryStorage) GetNotificationsStorage() *NotificationsMemoryStorage {
	return m.notifications
}

// NotificationsStorage methods - delegate to embedded notifications storage

func (m *MemoryStorage) CreateNotification(ctx context.Context, n *storage.Notification) error {
	return m.notifications.CreateNotification(ctx, n)
}

func (m *MemoryStorage) ListNotifications(ctx context.Context, profileID uuid.UUID, onlyUnread bool, limit, offset int) ([]storage.Notification, error) {
	return m.notifications.ListNotifications(ctx, profileID, onlyUnread, limit, offset)
}

func (m *MemoryStorage) UnreadCount(ctx context.Context, profileID uuid.UUID) (int, error) {
	return m.notifications.UnreadCount(ctx, profileID)
}

func (m *MemoryStorage) MarkRead(ctx context.Context, profileID uuid.UUID, ids []uuid.UUID) (int, error) {
	return m.notifications.MarkRead(ctx, profileID, ids)
}

func (m *MemoryStorage) MarkAllRead(ctx context.Context, profileID uuid.UUID) (int, error) {
	return m.notifications.MarkAllRead(ctx, profileID)
}

// GetSupplementsStorage returns the supplements storage
func (m *MemoryStorage) GetSupplementsStorage() *SupplementsMemoryStorage {
	return m.supplements
}

// GetIntakesStorage returns the intakes storage
func (m *MemoryStorage) GetIntakesStorage() *IntakesMemoryStorage {
	return m.intakes
}

// GetSupplementSchedulesStorage returns the supplement schedules storage.
func (m *MemoryStorage) GetSupplementSchedulesStorage() *SupplementSchedulesMemoryStorage {
	return m.schedules
}

// SupplementsStorage methods - delegate to embedded supplements storage

func (m *MemoryStorage) CreateSupplement(ctx context.Context, supplement *storage.Supplement) error {
	return m.supplements.CreateSupplement(ctx, supplement)
}

func (m *MemoryStorage) GetSupplement(ctx context.Context, id uuid.UUID) (*storage.Supplement, error) {
	return m.supplements.GetSupplement(ctx, id)
}

func (m *MemoryStorage) ListSupplements(ctx context.Context, profileID uuid.UUID) ([]storage.Supplement, error) {
	return m.supplements.ListSupplements(ctx, profileID)
}

func (m *MemoryStorage) UpdateSupplement(ctx context.Context, supplement *storage.Supplement) error {
	return m.supplements.UpdateSupplement(ctx, supplement)
}

func (m *MemoryStorage) DeleteSupplement(ctx context.Context, id uuid.UUID) error {
	return m.supplements.DeleteSupplement(ctx, id)
}

func (m *MemoryStorage) GetSupplementComponents(ctx context.Context, supplementID uuid.UUID) ([]storage.SupplementComponent, error) {
	return m.supplements.GetSupplementComponents(ctx, supplementID)
}

func (m *MemoryStorage) SetSupplementComponents(ctx context.Context, supplementID uuid.UUID, components []storage.SupplementComponent) error {
	return m.supplements.SetSupplementComponents(ctx, supplementID, components)
}

// GetEmailOTPStorage returns email OTP storage.
func (m *MemoryStorage) GetEmailOTPStorage() *EmailOTPMemoryStorage {
	return m.emailOTPs
}

// EmailOTPStorage methods - delegate to embedded email OTP storage.

func (m *MemoryStorage) CreateOrReplace(ctx context.Context, email, codeHash string, expiresAt, now time.Time, maxAttempts int) (uuid.UUID, error) {
	return m.emailOTPs.CreateOrReplace(ctx, email, codeHash, expiresAt, now, maxAttempts)
}

func (m *MemoryStorage) GetLatestActive(ctx context.Context, email string, now time.Time) (*storage.EmailOTP, error) {
	return m.emailOTPs.GetLatestActive(ctx, email, now)
}

func (m *MemoryStorage) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	return m.emailOTPs.IncrementAttempts(ctx, id)
}

func (m *MemoryStorage) MarkUsedOrDelete(ctx context.Context, id uuid.UUID) error {
	return m.emailOTPs.MarkUsedOrDelete(ctx, id)
}

func (m *MemoryStorage) UpdateResendMeta(ctx context.Context, id uuid.UUID, lastSentAt time.Time, sendCount int) error {
	return m.emailOTPs.UpdateResendMeta(ctx, id, lastSentAt, sendCount)
}

// GetSettingsStorage returns settings storage.
func (m *MemoryStorage) GetSettingsStorage() *SettingsMemoryStorage {
	return m.settings
}

// GetChatStorage returns chat storage.
func (m *MemoryStorage) GetChatStorage() *ChatMemoryStorage {
	return m.chat
}

// GetProposalsStorage returns proposals storage.
func (m *MemoryStorage) GetProposalsStorage() *ProposalsMemoryStorage {
	return m.proposals
}

// SettingsStorage methods - delegate to embedded settings storage.

func (m *MemoryStorage) GetSettings(ctx context.Context, ownerUserID string) (storage.Settings, bool, error) {
	return m.settings.GetSettings(ctx, ownerUserID)
}

func (m *MemoryStorage) UpsertSettings(ctx context.Context, ownerUserID string, s storage.Settings) (storage.Settings, error) {
	return m.settings.UpsertSettings(ctx, ownerUserID, s)
}

// ChatStorage methods - delegate to embedded chat storage.
func (m *MemoryStorage) InsertMessage(ctx context.Context, ownerUserID string, profileID uuid.UUID, role, content string) (storage.ChatMessage, error) {
	return m.chat.InsertMessage(ctx, ownerUserID, profileID, role, content)
}

func (m *MemoryStorage) ListMessages(ctx context.Context, ownerUserID string, profileID uuid.UUID, limit int, before *time.Time) ([]storage.ChatMessage, *time.Time, error) {
	return m.chat.ListMessages(ctx, ownerUserID, profileID, limit, before)
}

// ProposalsStorage methods - delegate to embedded proposals storage.
func (m *MemoryStorage) InsertMany(ctx context.Context, ownerUserID string, profileID uuid.UUID, drafts []storage.ProposalDraft) ([]storage.AIProposal, error) {
	return m.proposals.InsertMany(ctx, ownerUserID, profileID, drafts)
}

func (m *MemoryStorage) Get(ctx context.Context, ownerUserID string, proposalID uuid.UUID) (storage.AIProposal, bool, error) {
	return m.proposals.Get(ctx, ownerUserID, proposalID)
}

func (m *MemoryStorage) UpdateStatus(ctx context.Context, ownerUserID string, proposalID uuid.UUID, status string) error {
	return m.proposals.UpdateStatus(ctx, ownerUserID, proposalID, status)
}

func (m *MemoryStorage) List(ctx context.Context, ownerUserID string, profileID uuid.UUID, status string, limit int) ([]storage.AIProposal, error) {
	return m.proposals.List(ctx, ownerUserID, profileID, status, limit)
}

// IntakesStorage methods - delegate to embedded intakes storage

func (m *MemoryStorage) AddWater(ctx context.Context, profileID uuid.UUID, takenAt time.Time, amountMl int) error {
	return m.intakes.AddWater(ctx, profileID, takenAt, amountMl)
}

func (m *MemoryStorage) GetWaterDaily(ctx context.Context, profileID uuid.UUID, date string) (int, error) {
	return m.intakes.GetWaterDaily(ctx, profileID, date)
}

func (m *MemoryStorage) ListWaterIntakes(ctx context.Context, profileID uuid.UUID, date string, limit int) ([]storage.WaterIntake, error) {
	return m.intakes.ListWaterIntakes(ctx, profileID, date, limit)
}

func (m *MemoryStorage) UpsertSupplementIntake(ctx context.Context, intake *storage.SupplementIntake) error {
	return m.intakes.UpsertSupplementIntake(ctx, intake)
}

func (m *MemoryStorage) ListSupplementIntakes(ctx context.Context, profileID uuid.UUID, from, to string) ([]storage.SupplementIntake, error) {
	return m.intakes.ListSupplementIntakes(ctx, profileID, from, to)
}

func (m *MemoryStorage) GetSupplementDaily(ctx context.Context, profileID uuid.UUID, date string) (map[uuid.UUID]string, error) {
	return m.intakes.GetSupplementDaily(ctx, profileID, date)
}

// SupplementSchedulesStorage methods - delegate to embedded schedules storage.

func (m *MemoryStorage) ListSchedules(ctx context.Context, ownerUserID string, profileID uuid.UUID) ([]storage.SupplementSchedule, error) {
	return m.schedules.ListSchedules(ctx, ownerUserID, profileID)
}

func (m *MemoryStorage) UpsertSchedule(ctx context.Context, ownerUserID string, profileID uuid.UUID, item storage.ScheduleUpsert) (storage.SupplementSchedule, error) {
	return m.schedules.UpsertSchedule(ctx, ownerUserID, profileID, item)
}

func (m *MemoryStorage) DeleteSchedule(ctx context.Context, ownerUserID string, scheduleID uuid.UUID) error {
	return m.schedules.DeleteSchedule(ctx, ownerUserID, scheduleID)
}

func (m *MemoryStorage) ReplaceAll(ctx context.Context, ownerUserID string, profileID uuid.UUID, items []storage.ScheduleUpsert) ([]storage.SupplementSchedule, error) {
	return m.schedules.ReplaceAll(ctx, ownerUserID, profileID, items)
}

// GetWorkoutPlansStorage returns workout plans storage.
func (m *MemoryStorage) GetWorkoutPlansStorage() *WorkoutPlansStorage {
	return m.workoutPlans
}

// GetWorkoutPlanItemsStorage returns workout plan items storage.
func (m *MemoryStorage) GetWorkoutPlanItemsStorage() *WorkoutPlanItemsStorage {
	return m.workoutItems
}

// GetWorkoutCompletionsStorage returns workout completions storage.
func (m *MemoryStorage) GetWorkoutCompletionsStorage() *WorkoutCompletionsStorage {
	return m.workoutCompletions
}

// GetNutritionTargetsStorage returns nutrition targets storage
func (m *MemoryStorage) GetNutritionTargetsStorage() storage.NutritionTargetsStorage {
	return m.nutritionTargets
}

// WorkoutPlansStorage methods - delegate to embedded workout plans storage.

func (m *MemoryStorage) GetActivePlan(ownerUserID string, profileID uuid.UUID) (storage.WorkoutPlan, bool, error) {
	return m.workoutPlans.GetActivePlan(ownerUserID, profileID)
}

func (m *MemoryStorage) UpsertActivePlan(ownerUserID string, profileID uuid.UUID, title string, goal string) (storage.WorkoutPlan, error) {
	return m.workoutPlans.UpsertActivePlan(ownerUserID, profileID, title, goal)
}

// WorkoutPlanItemsStorage methods - delegate to embedded workout plan items storage.

func (m *MemoryStorage) ListItems(ownerUserID string, profileID uuid.UUID, planID uuid.UUID) ([]storage.WorkoutPlanItem, error) {
	return m.workoutItems.ListItems(ownerUserID, profileID, planID)
}

func (m *MemoryStorage) ReplaceAllItems(ownerUserID string, profileID uuid.UUID, planID uuid.UUID, items []storage.WorkoutItemUpsert) ([]storage.WorkoutPlanItem, error) {
	return m.workoutItems.ReplaceAllItems(ownerUserID, profileID, planID, items)
}

func (m *MemoryStorage) DeleteItem(ownerUserID string, itemID uuid.UUID) error {
	return m.workoutItems.DeleteItem(ownerUserID, itemID)
}

// WorkoutCompletionsStorage methods - delegate to embedded workout completions storage.

func (m *MemoryStorage) UpsertCompletion(ownerUserID string, profileID uuid.UUID, date string, planItemID uuid.UUID, status string, note string) (storage.WorkoutCompletion, error) {
	return m.workoutCompletions.UpsertCompletion(ownerUserID, profileID, date, planItemID, status, note)
}

func (m *MemoryStorage) ListCompletions(ownerUserID string, profileID uuid.UUID, from string, to string) ([]storage.WorkoutCompletion, error) {
	return m.workoutCompletions.ListCompletions(ownerUserID, profileID, from, to)
}

// FoodPrefsStorage methods - delegate to embedded food prefs storage.

func (m *MemoryStorage) GetFoodPrefsStorage() storage.FoodPrefsStorage {
	return m.foodPrefs
}

// MealPlansStorage methods - delegate to embedded meal plans storage.

func (m *MemoryStorage) GetMealPlansStorage() storage.MealPlansStorage {
	return m.mealPlans
}
