package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("profile not found")
)

// PostgresStorage — Postgres реализация Storage и MetрicsStorage
type PostgresStorage struct {
	pool               *pgxpool.Pool
	metrics            *PostgresMetricsStorage
	checkins           *PostgresCheckinsStorage
	reports            *PostgresReportsStorage
	sources            *PostgresSourcesStorage
	notifications      *PostgresNotificationsStorage
	supplements        *PostgresSupplementsStorage
	schedules          *PostgresSupplementSchedulesStorage
	intakes            *PostgresIntakesStorage
	emailOTPs          *PostgresEmailOTPStorage
	settings           *PostgresSettingsStorage
	chat               *PostgresChatStorage
	proposals          *PostgresProposalsStorage
	workoutPlans       *PostgresWorkoutPlansStorage
	workoutItems       *PostgresWorkoutPlanItemsStorage
	workoutCompletions *PostgresWorkoutCompletionsStorage
	nutritionTargets   *nutritionTargetsStorage
	foodPrefs          *foodPrefsStorage
	mealPlans          *mealPlansStorage
}

// New создаёт PostgresStorage и обеспечивает owner профиль по умолчанию
func New(ctx context.Context, databaseURL string) (*PostgresStorage, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	ps := &PostgresStorage{
		pool:               pool,
		metrics:            NewMetricsStorage(pool),
		checkins:           NewPostgresCheckinsStorage(pool),
		reports:            NewPostgresReportsStorage(pool),
		sources:            NewPostgresSourcesStorage(pool),
		notifications:      NewPostgresNotificationsStorage(pool),
		supplements:        NewPostgresSupplementsStorage(pool),
		schedules:          NewPostgresSupplementSchedulesStorage(pool),
		intakes:            NewPostgresIntakesStorage(pool),
		emailOTPs:          NewPostgresEmailOTPStorage(pool),
		settings:           NewPostgresSettingsStorage(pool),
		chat:               NewPostgresChatStorage(pool),
		proposals:          NewPostgresProposalsStorage(pool),
		workoutPlans:       NewPostgresWorkoutPlansStorage(pool),
		workoutItems:       NewPostgresWorkoutPlanItemsStorage(pool),
		workoutCompletions: NewPostgresWorkoutCompletionsStorage(pool),
		nutritionTargets:   newNutritionTargetsStorage(pool),
		foodPrefs:          newFoodPrefsStorage(pool),
		mealPlans:          newMealPlansStorage(pool),
	}

	// Создаём owner профиль, если его нет
	if err := ps.ensureOwnerProfile(ctx); err != nil {
		return nil, err
	}

	return ps, nil
}

// ensureOwnerProfile создаёт owner профиль, если его ещё нет
func (p *PostgresStorage) ensureOwnerProfile(ctx context.Context) error {
	query := `
		INSERT INTO profiles (id, owner_user_id, type, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT DO NOTHING
	`

	ownerID := uuid.New()
	now := time.Now()

	_, err := p.pool.Exec(ctx, query,
		ownerID,
		"default",
		"owner",
		"Я",
		now,
		now,
	)

	return err
}

func (p *PostgresStorage) ListProfiles(ctx context.Context) ([]storage.Profile, error) {
	query := `
		SELECT id, owner_user_id, type, name, created_at, updated_at
		FROM profiles
		ORDER BY created_at ASC
	`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := []storage.Profile{}
	for rows.Next() {
		var prof storage.Profile
		err := rows.Scan(
			&prof.ID,
			&prof.OwnerUserID,
			&prof.Type,
			&prof.Name,
			&prof.CreatedAt,
			&prof.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, prof)
	}

	return profiles, rows.Err()
}

func (p *PostgresStorage) GetProfile(ctx context.Context, id uuid.UUID) (*storage.Profile, error) {
	query := `
		SELECT id, owner_user_id, type, name, created_at, updated_at
		FROM profiles
		WHERE id = $1
	`

	var prof storage.Profile
	err := p.pool.QueryRow(ctx, query, id).Scan(
		&prof.ID,
		&prof.OwnerUserID,
		&prof.Type,
		&prof.Name,
		&prof.CreatedAt,
		&prof.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	return &prof, nil
}

func (p *PostgresStorage) CreateProfile(ctx context.Context, profile *storage.Profile) error {
	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}

	now := time.Now()
	profile.CreatedAt = now
	profile.UpdatedAt = now

	query := `
		INSERT INTO profiles (id, owner_user_id, type, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := p.pool.Exec(ctx, query,
		profile.ID,
		profile.OwnerUserID,
		profile.Type,
		profile.Name,
		profile.CreatedAt,
		profile.UpdatedAt,
	)

	return err
}

func (p *PostgresStorage) UpdateProfile(ctx context.Context, profile *storage.Profile) error {
	profile.UpdatedAt = time.Now()

	query := `
		UPDATE profiles
		SET name = $2, updated_at = $3
		WHERE id = $1
	`

	result, err := p.pool.Exec(ctx, query,
		profile.ID,
		profile.Name,
		profile.UpdatedAt,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (p *PostgresStorage) DeleteProfile(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM profiles WHERE id = $1`

	result, err := p.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (p *PostgresStorage) Close() error {
	p.pool.Close()
	return nil
}

// MetricsStorage methods - делегируем к встроенному metrics storage

func (p *PostgresStorage) UpsertDailyMetric(ctx context.Context, profileID uuid.UUID, date string, payload []byte) error {
	return p.metrics.UpsertDailyMetric(ctx, profileID, date, payload)
}

func (p *PostgresStorage) GetDailyMetrics(ctx context.Context, profileID uuid.UUID, from, to string) ([]storage.DailyMetricRow, error) {
	return p.metrics.GetDailyMetrics(ctx, profileID, from, to)
}

func (p *PostgresStorage) UpsertHourlyMetric(ctx context.Context, profileID uuid.UUID, hour time.Time, steps *int, hrMin, hrMax, hrAvg *int) error {
	return p.metrics.UpsertHourlyMetric(ctx, profileID, hour, steps, hrMin, hrMax, hrAvg)
}

func (p *PostgresStorage) GetHourlyMetrics(ctx context.Context, profileID uuid.UUID, date string) ([]storage.HourlyMetricRow, error) {
	return p.metrics.GetHourlyMetrics(ctx, profileID, date)
}

func (p *PostgresStorage) InsertSleepSegment(ctx context.Context, profileID uuid.UUID, start, end time.Time, stage string) error {
	return p.metrics.InsertSleepSegment(ctx, profileID, start, end, stage)
}

func (p *PostgresStorage) InsertWorkout(ctx context.Context, profileID uuid.UUID, start, end time.Time, label string, caloriesKcal *int) error {
	return p.metrics.InsertWorkout(ctx, profileID, start, end, label, caloriesKcal)
}

// GetCheckinsStorage returns the checkins storage
func (p *PostgresStorage) GetCheckinsStorage() *PostgresCheckinsStorage {
	return p.checkins
}

// GetReportsStorage returns the reports storage
func (p *PostgresStorage) GetReportsStorage() *PostgresReportsStorage {
	return p.reports
}

// GetSourcesStorage returns the sources storage
func (p *PostgresStorage) GetSourcesStorage() *PostgresSourcesStorage {
	return p.sources
}

// SourcesStorage methods - delegate to embedded sources storage

func (p *PostgresStorage) CreateSource(ctx context.Context, source *storage.Source) error {
	return p.sources.CreateSource(ctx, source)
}

func (p *PostgresStorage) GetSource(ctx context.Context, id uuid.UUID) (*storage.Source, error) {
	return p.sources.GetSource(ctx, id)
}

func (p *PostgresStorage) ListSources(ctx context.Context, profileID uuid.UUID, query string, checkinID *uuid.UUID, limit, offset int) ([]storage.Source, error) {
	return p.sources.ListSources(ctx, profileID, query, checkinID, limit, offset)
}

func (p *PostgresStorage) DeleteSource(ctx context.Context, id uuid.UUID) error {
	return p.sources.DeleteSource(ctx, id)
}

func (p *PostgresStorage) GetSourceBlob(ctx context.Context, sourceID uuid.UUID) ([]byte, string, error) {
	return p.sources.GetSourceBlob(ctx, sourceID)
}

func (p *PostgresStorage) PutSourceBlob(ctx context.Context, sourceID uuid.UUID, data []byte, contentType string) error {
	return p.sources.PutSourceBlob(ctx, sourceID, data, contentType)
}

// GetNotificationsStorage returns the notifications storage
func (p *PostgresStorage) GetNotificationsStorage() *PostgresNotificationsStorage {
	return p.notifications
}

// NotificationsStorage methods - delegate to embedded notifications storage

func (p *PostgresStorage) CreateNotification(ctx context.Context, n *storage.Notification) error {
	return p.notifications.CreateNotification(ctx, n)
}

func (p *PostgresStorage) ListNotifications(ctx context.Context, profileID uuid.UUID, onlyUnread bool, limit, offset int) ([]storage.Notification, error) {
	return p.notifications.ListNotifications(ctx, profileID, onlyUnread, limit, offset)
}

func (p *PostgresStorage) UnreadCount(ctx context.Context, profileID uuid.UUID) (int, error) {
	return p.notifications.UnreadCount(ctx, profileID)
}

func (p *PostgresStorage) MarkRead(ctx context.Context, profileID uuid.UUID, ids []uuid.UUID) (int, error) {
	return p.notifications.MarkRead(ctx, profileID, ids)
}

func (p *PostgresStorage) MarkAllRead(ctx context.Context, profileID uuid.UUID) (int, error) {
	return p.notifications.MarkAllRead(ctx, profileID)
}

// GetSupplementsStorage returns the supplements storage
func (p *PostgresStorage) GetSupplementsStorage() *PostgresSupplementsStorage {
	return p.supplements
}

// GetIntakesStorage returns the intakes storage
func (p *PostgresStorage) GetIntakesStorage() *PostgresIntakesStorage {
	return p.intakes
}

// GetSupplementSchedulesStorage returns the supplement schedules storage.
func (p *PostgresStorage) GetSupplementSchedulesStorage() *PostgresSupplementSchedulesStorage {
	return p.schedules
}

// SupplementsStorage methods - delegate to embedded supplements storage

func (p *PostgresStorage) CreateSupplement(ctx context.Context, supplement *storage.Supplement) error {
	return p.supplements.CreateSupplement(ctx, supplement)
}

func (p *PostgresStorage) GetSupplement(ctx context.Context, id uuid.UUID) (*storage.Supplement, error) {
	return p.supplements.GetSupplement(ctx, id)
}

func (p *PostgresStorage) ListSupplements(ctx context.Context, profileID uuid.UUID) ([]storage.Supplement, error) {
	return p.supplements.ListSupplements(ctx, profileID)
}

func (p *PostgresStorage) UpdateSupplement(ctx context.Context, supplement *storage.Supplement) error {
	return p.supplements.UpdateSupplement(ctx, supplement)
}

func (p *PostgresStorage) DeleteSupplement(ctx context.Context, id uuid.UUID) error {
	return p.supplements.DeleteSupplement(ctx, id)
}

func (p *PostgresStorage) GetSupplementComponents(ctx context.Context, supplementID uuid.UUID) ([]storage.SupplementComponent, error) {
	return p.supplements.GetSupplementComponents(ctx, supplementID)
}

func (p *PostgresStorage) SetSupplementComponents(ctx context.Context, supplementID uuid.UUID, components []storage.SupplementComponent) error {
	return p.supplements.SetSupplementComponents(ctx, supplementID, components)
}

// GetEmailOTPStorage returns email OTP storage.
func (p *PostgresStorage) GetEmailOTPStorage() *PostgresEmailOTPStorage {
	return p.emailOTPs
}

// EmailOTPStorage methods - delegate to embedded email OTP storage.

func (p *PostgresStorage) CreateOrReplace(ctx context.Context, email, codeHash string, expiresAt, now time.Time, maxAttempts int) (uuid.UUID, error) {
	return p.emailOTPs.CreateOrReplace(ctx, email, codeHash, expiresAt, now, maxAttempts)
}

func (p *PostgresStorage) GetLatestActive(ctx context.Context, email string, now time.Time) (*storage.EmailOTP, error) {
	return p.emailOTPs.GetLatestActive(ctx, email, now)
}

func (p *PostgresStorage) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	return p.emailOTPs.IncrementAttempts(ctx, id)
}

func (p *PostgresStorage) MarkUsedOrDelete(ctx context.Context, id uuid.UUID) error {
	return p.emailOTPs.MarkUsedOrDelete(ctx, id)
}

func (p *PostgresStorage) UpdateResendMeta(ctx context.Context, id uuid.UUID, lastSentAt time.Time, sendCount int) error {
	return p.emailOTPs.UpdateResendMeta(ctx, id, lastSentAt, sendCount)
}

// GetSettingsStorage returns settings storage.
func (p *PostgresStorage) GetSettingsStorage() *PostgresSettingsStorage {
	return p.settings
}

// GetChatStorage returns chat storage.
func (p *PostgresStorage) GetChatStorage() *PostgresChatStorage {
	return p.chat
}

// GetProposalsStorage returns proposals storage.
func (p *PostgresStorage) GetProposalsStorage() *PostgresProposalsStorage {
	return p.proposals
}

// SettingsStorage methods - delegate to embedded settings storage.

func (p *PostgresStorage) GetSettings(ctx context.Context, ownerUserID string) (storage.Settings, bool, error) {
	return p.settings.GetSettings(ctx, ownerUserID)
}

func (p *PostgresStorage) UpsertSettings(ctx context.Context, ownerUserID string, s storage.Settings) (storage.Settings, error) {
	return p.settings.UpsertSettings(ctx, ownerUserID, s)
}

// ChatStorage methods - delegate to embedded chat storage.
func (p *PostgresStorage) InsertMessage(ctx context.Context, ownerUserID string, profileID uuid.UUID, role, content string) (storage.ChatMessage, error) {
	return p.chat.InsertMessage(ctx, ownerUserID, profileID, role, content)
}

func (p *PostgresStorage) ListMessages(ctx context.Context, ownerUserID string, profileID uuid.UUID, limit int, before *time.Time) ([]storage.ChatMessage, *time.Time, error) {
	return p.chat.ListMessages(ctx, ownerUserID, profileID, limit, before)
}

// ProposalsStorage methods - delegate to embedded proposals storage.
func (p *PostgresStorage) InsertMany(ctx context.Context, ownerUserID string, profileID uuid.UUID, drafts []storage.ProposalDraft) ([]storage.AIProposal, error) {
	return p.proposals.InsertMany(ctx, ownerUserID, profileID, drafts)
}

func (p *PostgresStorage) Get(ctx context.Context, ownerUserID string, proposalID uuid.UUID) (storage.AIProposal, bool, error) {
	return p.proposals.Get(ctx, ownerUserID, proposalID)
}

func (p *PostgresStorage) UpdateStatus(ctx context.Context, ownerUserID string, proposalID uuid.UUID, status string) error {
	return p.proposals.UpdateStatus(ctx, ownerUserID, proposalID, status)
}

func (p *PostgresStorage) List(ctx context.Context, ownerUserID string, profileID uuid.UUID, status string, limit int) ([]storage.AIProposal, error) {
	return p.proposals.List(ctx, ownerUserID, profileID, status, limit)
}

// IntakesStorage methods - delegate to embedded intakes storage

func (p *PostgresStorage) AddWater(ctx context.Context, profileID uuid.UUID, takenAt time.Time, amountMl int) error {
	return p.intakes.AddWater(ctx, profileID, takenAt, amountMl)
}

func (p *PostgresStorage) GetWaterDaily(ctx context.Context, profileID uuid.UUID, date string) (int, error) {
	return p.intakes.GetWaterDaily(ctx, profileID, date)
}

func (p *PostgresStorage) ListWaterIntakes(ctx context.Context, profileID uuid.UUID, date string, limit int) ([]storage.WaterIntake, error) {
	return p.intakes.ListWaterIntakes(ctx, profileID, date, limit)
}

func (p *PostgresStorage) UpsertSupplementIntake(ctx context.Context, intake *storage.SupplementIntake) error {
	return p.intakes.UpsertSupplementIntake(ctx, intake)
}

func (p *PostgresStorage) ListSupplementIntakes(ctx context.Context, profileID uuid.UUID, from, to string) ([]storage.SupplementIntake, error) {
	return p.intakes.ListSupplementIntakes(ctx, profileID, from, to)
}

func (p *PostgresStorage) GetSupplementDaily(ctx context.Context, profileID uuid.UUID, date string) (map[uuid.UUID]string, error) {
	return p.intakes.GetSupplementDaily(ctx, profileID, date)
}

// SupplementSchedulesStorage methods - delegate to embedded schedules storage.

func (p *PostgresStorage) ListSchedules(ctx context.Context, ownerUserID string, profileID uuid.UUID) ([]storage.SupplementSchedule, error) {
	return p.schedules.ListSchedules(ctx, ownerUserID, profileID)
}

func (p *PostgresStorage) UpsertSchedule(ctx context.Context, ownerUserID string, profileID uuid.UUID, item storage.ScheduleUpsert) (storage.SupplementSchedule, error) {
	return p.schedules.UpsertSchedule(ctx, ownerUserID, profileID, item)
}

func (p *PostgresStorage) DeleteSchedule(ctx context.Context, ownerUserID string, scheduleID uuid.UUID) error {
	return p.schedules.DeleteSchedule(ctx, ownerUserID, scheduleID)
}

func (p *PostgresStorage) ReplaceAll(ctx context.Context, ownerUserID string, profileID uuid.UUID, items []storage.ScheduleUpsert) ([]storage.SupplementSchedule, error) {
	return p.schedules.ReplaceAll(ctx, ownerUserID, profileID, items)
}

// GetWorkoutPlansStorage returns workout plans storage.
func (p *PostgresStorage) GetWorkoutPlansStorage() *PostgresWorkoutPlansStorage {
	return p.workoutPlans
}

// GetWorkoutPlanItemsStorage returns workout plan items storage.
func (p *PostgresStorage) GetWorkoutPlanItemsStorage() *PostgresWorkoutPlanItemsStorage {
	return p.workoutItems
}

// GetWorkoutCompletionsStorage returns workout completions storage.
func (p *PostgresStorage) GetWorkoutCompletionsStorage() *PostgresWorkoutCompletionsStorage {
	return p.workoutCompletions
}

// GetNutritionTargetsStorage returns nutrition targets storage
func (p *PostgresStorage) GetNutritionTargetsStorage() storage.NutritionTargetsStorage {
	return p.nutritionTargets
}

// WorkoutPlansStorage methods - delegate to embedded workout plans storage.

func (p *PostgresStorage) GetActivePlan(ownerUserID string, profileID uuid.UUID) (storage.WorkoutPlan, bool, error) {
	return p.workoutPlans.GetActivePlan(ownerUserID, profileID)
}

func (p *PostgresStorage) UpsertActivePlan(ownerUserID string, profileID uuid.UUID, title string, goal string) (storage.WorkoutPlan, error) {
	return p.workoutPlans.UpsertActivePlan(ownerUserID, profileID, title, goal)
}

// WorkoutPlanItemsStorage methods - delegate to embedded workout plan items storage.

func (p *PostgresStorage) ListItems(ownerUserID string, profileID uuid.UUID, planID uuid.UUID) ([]storage.WorkoutPlanItem, error) {
	return p.workoutItems.ListItems(ownerUserID, profileID, planID)
}

func (p *PostgresStorage) ReplaceAllItems(ownerUserID string, profileID uuid.UUID, planID uuid.UUID, items []storage.WorkoutItemUpsert) ([]storage.WorkoutPlanItem, error) {
	return p.workoutItems.ReplaceAllItems(ownerUserID, profileID, planID, items)
}

func (p *PostgresStorage) DeleteItem(ownerUserID string, itemID uuid.UUID) error {
	return p.workoutItems.DeleteItem(ownerUserID, itemID)
}

// WorkoutCompletionsStorage methods - delegate to embedded workout completions storage.

func (p *PostgresStorage) UpsertCompletion(ownerUserID string, profileID uuid.UUID, date string, planItemID uuid.UUID, status string, note string) (storage.WorkoutCompletion, error) {
	return p.workoutCompletions.UpsertCompletion(ownerUserID, profileID, date, planItemID, status, note)
}

func (p *PostgresStorage) ListCompletions(ownerUserID string, profileID uuid.UUID, from string, to string) ([]storage.WorkoutCompletion, error) {
	return p.workoutCompletions.ListCompletions(ownerUserID, profileID, from, to)
}

// FoodPrefsStorage methods - delegate to embedded food prefs storage.

func (p *PostgresStorage) GetFoodPrefsStorage() storage.FoodPrefsStorage {
	return p.foodPrefs
}

// MealPlansStorage methods - delegate to embedded meal plans storage.

func (p *PostgresStorage) GetMealPlansStorage() storage.MealPlansStorage {
	return p.mealPlans
}
