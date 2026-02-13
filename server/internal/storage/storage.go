package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Profile представляет профиль пользователя (owner или guest)
type Profile struct {
	ID          uuid.UUID
	OwnerUserID string // "default" для MVP, позже uuid
	Type        string // "owner" или "guest"
	Name        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Storage — интерфейс для работы с профилями
type Storage interface {
	// ListProfiles возвращает все профили
	ListProfiles(ctx context.Context) ([]Profile, error)

	// GetProfile возвращает профиль по ID
	GetProfile(ctx context.Context, id uuid.UUID) (*Profile, error)

	// CreateProfile создаёт новый профиль
	CreateProfile(ctx context.Context, profile *Profile) error

	// UpdateProfile обновляет профиль
	UpdateProfile(ctx context.Context, profile *Profile) error

	// DeleteProfile удаляет профиль
	DeleteProfile(ctx context.Context, id uuid.UUID) error

	// Close закрывает соединение (для Postgres)
	Close() error
}

// MetricsStorage — интерфейс для работы с метриками
type MetricsStorage interface {
	// UpsertDailyMetric сохраняет дневную метрику (upsert по profile_id, date)
	UpsertDailyMetric(ctx context.Context, profileID uuid.UUID, date string, payload []byte) error

	// GetDailyMetrics возвращает дневные метрики за период
	GetDailyMetrics(ctx context.Context, profileID uuid.UUID, from, to string) ([]DailyMetricRow, error)

	// UpsertHourlyMetric сохраняет часовую метрику (upsert по profile_id, hour)
	UpsertHourlyMetric(ctx context.Context, profileID uuid.UUID, hour time.Time, steps *int, hrMin, hrMax, hrAvg *int) error

	// GetHourlyMetrics возвращает часовые метрики за день
	GetHourlyMetrics(ctx context.Context, profileID uuid.UUID, date string) ([]HourlyMetricRow, error)

	// InsertSleepSegment добавляет сегмент сна (ignore duplicates)
	InsertSleepSegment(ctx context.Context, profileID uuid.UUID, start, end time.Time, stage string) error

	// InsertWorkout добавляет тренировку (ignore duplicates)
	InsertWorkout(ctx context.Context, profileID uuid.UUID, start, end time.Time, label string, caloriesKcal *int) error
}

// DailyMetricRow — строка из daily_metrics
type DailyMetricRow struct {
	ProfileID uuid.UUID
	Date      string // YYYY-MM-DD
	Payload   []byte // JSON
	CreatedAt time.Time
	UpdatedAt time.Time
}

// HourlyMetricRow — строка из hourly_metrics
type HourlyMetricRow struct {
	ProfileID uuid.UUID
	Hour      time.Time
	Steps     *int
	HRMin     *int
	HRMax     *int
	HRAvg     *int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ReportsStorage — интерфейс для работы с отчётами
type ReportsStorage interface {
	// CreateReport создаёт новый отчёт (metadata + optional data for memory mode)
	CreateReport(ctx context.Context, report *ReportMeta) error

	// GetReport возвращает отчёт по ID
	GetReport(ctx context.Context, id uuid.UUID) (*ReportMeta, error)

	// ListReports возвращает список отчётов профиля с пагинацией
	ListReports(ctx context.Context, profileID uuid.UUID, limit, offset int) ([]ReportMeta, error)

	// DeleteReport удаляет отчёт (metadata и данные)
	DeleteReport(ctx context.Context, id uuid.UUID) error
}

// ReportMeta — метаданные отчёта
type ReportMeta struct {
	ID        uuid.UUID
	ProfileID uuid.UUID
	Format    string  // "pdf" or "csv"
	FromDate  string  // YYYY-MM-DD
	ToDate    string  // YYYY-MM-DD
	ObjectKey *string // S3 object key (NULL for memory mode)
	SizeBytes int64
	Status    string // "ready" or "failed"
	Error     *string
	CreatedAt time.Time
	UpdatedAt time.Time
	Data      []byte // Only used in memory mode (not stored in DB)
}

// SourcesStorage — интерфейс для работы с sources (links, notes, images)
type SourcesStorage interface {
	// CreateSource создаёт новый source
	CreateSource(ctx context.Context, source *Source) error

	// GetSource возвращает source по ID
	GetSource(ctx context.Context, id uuid.UUID) (*Source, error)

	// ListSources возвращает список sources для профиля с опциональной фильтрацией
	ListSources(ctx context.Context, profileID uuid.UUID, query string, checkinID *uuid.UUID, limit, offset int) ([]Source, error)

	// DeleteSource удаляет source
	DeleteSource(ctx context.Context, id uuid.UUID) error

	// GetSourceBlob возвращает blob данные для image source (memory mode only)
	GetSourceBlob(ctx context.Context, sourceID uuid.UUID) ([]byte, string, error)

	// PutSourceBlob сохраняет blob данные для image source (memory mode only)
	PutSourceBlob(ctx context.Context, sourceID uuid.UUID, data []byte, contentType string) error
}

// Source — пользовательский контент (link, note, image)
type Source struct {
	ID          uuid.UUID
	ProfileID   uuid.UUID
	Kind        string     // "link", "note", "image"
	Title       *string    // optional
	Text        *string    // for notes
	URL         *string    // for links
	CheckinID   *uuid.UUID // optional reference to checkin
	ObjectKey   *string    // S3 object key (images only, S3 mode)
	ContentType *string    // MIME type (images only)
	SizeBytes   int64      // file size (images only)
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NotificationsStorage — интерфейс для работы с notifications/inbox
type NotificationsStorage interface {
	// CreateNotification создаёт уведомление (upsert by unique key)
	CreateNotification(ctx context.Context, n *Notification) error

	// ListNotifications возвращает список уведомлений для профиля
	ListNotifications(ctx context.Context, profileID uuid.UUID, onlyUnread bool, limit, offset int) ([]Notification, error)

	// UnreadCount возвращает количество непрочитанных уведомлений
	UnreadCount(ctx context.Context, profileID uuid.UUID) (int, error)

	// MarkRead отмечает указанные уведомления как прочитанные (проверяет принадлежность профилю)
	MarkRead(ctx context.Context, profileID uuid.UUID, ids []uuid.UUID) (int, error)

	// MarkAllRead отмечает все уведомления профиля как прочитанные
	MarkAllRead(ctx context.Context, profileID uuid.UUID) (int, error)
}

// Notification — уведомление для пользователя
type Notification struct {
	ID         uuid.UUID
	ProfileID  uuid.UUID
	Kind       string // low_sleep, low_activity, missing_morning_checkin, missing_evening_checkin
	Title      string
	Body       string
	SourceDate *time.Time // date this notification relates to (nullable)
	Severity   string     // "info" or "warn"
	CreatedAt  time.Time
	ReadAt     *time.Time
}

// SupplementsStorage — интерфейс для работы с supplements
type SupplementsStorage interface {
	// CreateSupplement создаёт новую добавку
	CreateSupplement(ctx context.Context, supplement *Supplement) error

	// GetSupplement возвращает добавку по ID
	GetSupplement(ctx context.Context, id uuid.UUID) (*Supplement, error)

	// ListSupplements возвращает список добавок для профиля
	ListSupplements(ctx context.Context, profileID uuid.UUID) ([]Supplement, error)

	// UpdateSupplement обновляет добавку
	UpdateSupplement(ctx context.Context, supplement *Supplement) error

	// DeleteSupplement удаляет добавку
	DeleteSupplement(ctx context.Context, id uuid.UUID) error

	// GetSupplementComponents возвращает компоненты добавки
	GetSupplementComponents(ctx context.Context, supplementID uuid.UUID) ([]SupplementComponent, error)

	// SetSupplementComponents заменяет все компоненты добавки
	SetSupplementComponents(ctx context.Context, supplementID uuid.UUID, components []SupplementComponent) error
}

// IntakesStorage — интерфейс для работы с water/supplement intakes
type IntakesStorage interface {
	// AddWater добавляет запись о приёме воды
	AddWater(ctx context.Context, profileID uuid.UUID, takenAt time.Time, amountMl int) error

	// GetWaterDaily возвращает суммарное количество воды за день
	GetWaterDaily(ctx context.Context, profileID uuid.UUID, date string) (int, error)

	// ListWaterIntakes возвращает список приёмов воды за день
	ListWaterIntakes(ctx context.Context, profileID uuid.UUID, date string, limit int) ([]WaterIntake, error)

	// UpsertSupplementIntake создаёт/обновляет отметку о приёме добавки (upsert by unique key)
	UpsertSupplementIntake(ctx context.Context, intake *SupplementIntake) error

	// ListSupplementIntakes возвращает отметки о приёме добавок за период
	ListSupplementIntakes(ctx context.Context, profileID uuid.UUID, from, to string) ([]SupplementIntake, error)

	// GetSupplementDaily возвращает статусы добавок за день
	GetSupplementDaily(ctx context.Context, profileID uuid.UUID, date string) (map[uuid.UUID]string, error)
}

// SupplementSchedulesStorage — интерфейс для расписаний приёма добавок.
type SupplementSchedulesStorage interface {
	// ListSchedules возвращает расписания профиля (с учётом owner/profile).
	ListSchedules(ctx context.Context, ownerUserID string, profileID uuid.UUID) ([]SupplementSchedule, error)

	// UpsertSchedule создаёт или обновляет расписание по unique(owner, profile, supplement, time).
	UpsertSchedule(ctx context.Context, ownerUserID string, profileID uuid.UUID, item ScheduleUpsert) (SupplementSchedule, error)

	// DeleteSchedule удаляет расписание по id в рамках owner.
	DeleteSchedule(ctx context.Context, ownerUserID string, scheduleID uuid.UUID) error

	// ReplaceAll атомарно заменяет набор расписаний профиля в рамках owner/profile.
	ReplaceAll(ctx context.Context, ownerUserID string, profileID uuid.UUID, items []ScheduleUpsert) ([]SupplementSchedule, error)
}

// Supplement — добавка/витамин
type Supplement struct {
	ID        uuid.UUID
	ProfileID uuid.UUID
	Name      string
	Notes     *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SupplementComponent — компонент добавки
type SupplementComponent struct {
	ID           uuid.UUID
	SupplementID uuid.UUID
	NutrientKey  string
	HKIdentifier *string // HealthKit identifier (optional)
	Amount       float64
	Unit         string
	CreatedAt    time.Time
}

// WaterIntake — запись о приёме воды
type WaterIntake struct {
	ID        uuid.UUID
	ProfileID uuid.UUID
	TakenAt   time.Time
	AmountMl  int
	CreatedAt time.Time
}

// SupplementIntake — отметка о приёме добавки
type SupplementIntake struct {
	ID           uuid.UUID
	ProfileID    uuid.UUID
	SupplementID uuid.UUID
	TakenAt      time.Time
	Status       string // "taken" or "skipped"
	CreatedAt    time.Time
}

// SupplementSchedule — расписание приёма добавки.
type SupplementSchedule struct {
	ID           uuid.UUID
	OwnerUserID  string
	ProfileID    uuid.UUID
	SupplementID uuid.UUID
	TimeMinutes  int
	DaysMask     int
	IsEnabled    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ScheduleUpsert — входные данные для upsert/replace расписания.
type ScheduleUpsert struct {
	SupplementID uuid.UUID
	TimeMinutes  int
	DaysMask     int
	IsEnabled    bool
}

// EmailOTPStorage — интерфейс для работы с email OTP кодами.
type EmailOTPStorage interface {
	// CreateOrReplace создаёт новый активный OTP для email, заменяя прошлый активный.
	CreateOrReplace(ctx context.Context, email, codeHash string, expiresAt, now time.Time, maxAttempts int) (uuid.UUID, error)

	// GetLatestActive возвращает самый свежий неистёкший OTP.
	GetLatestActive(ctx context.Context, email string, now time.Time) (*EmailOTP, error)

	// IncrementAttempts увеличивает счётчик неудачных попыток.
	IncrementAttempts(ctx context.Context, id uuid.UUID) error

	// MarkUsedOrDelete помечает OTP использованным (для MVP удаляем запись).
	MarkUsedOrDelete(ctx context.Context, id uuid.UUID) error

	// UpdateResendMeta обновляет метаданные повторной отправки.
	UpdateResendMeta(ctx context.Context, id uuid.UUID, lastSentAt time.Time, sendCount int) error
}

// EmailOTP — запись OTP кода для email.
type EmailOTP struct {
	ID          uuid.UUID
	Email       string
	CodeHash    string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	Attempts    int
	MaxAttempts int
	LastSentAt  time.Time
	SendCount   int
}

// SettingsStorage — интерфейс для пользовательских настроек уведомлений/порогов.
type SettingsStorage interface {
	// GetSettings returns settings by owner_user_id. bool=false means not found.
	GetSettings(ctx context.Context, ownerUserID string) (Settings, bool, error)

	// UpsertSettings creates or updates settings for owner_user_id.
	UpsertSettings(ctx context.Context, ownerUserID string, s Settings) (Settings, error)
}

// Settings — persisted per-user settings.
type Settings struct {
	OwnerUserID string

	TimeZone *string

	QuietStartMinutes *int
	QuietEndMinutes   *int

	NotificationsMaxPerDay int

	MinSleepMinutes      int
	MinSteps             int
	MinActiveEnergyKcal  int
	MorningCheckinMinute int
	EveningCheckinMinute int
	VitaminsTimeMinute   int

	CreatedAt time.Time
	UpdatedAt time.Time
}

// ChatStorage — интерфейс для хранения сообщений чата.
type ChatStorage interface {
	// InsertMessage сохраняет сообщение чата.
	InsertMessage(ctx context.Context, ownerUserID string, profileID uuid.UUID, role, content string) (ChatMessage, error)

	// ListMessages возвращает последние сообщения по owner/profile и nextCursor.
	// before используется как курсор по created_at (strictly less than).
	ListMessages(ctx context.Context, ownerUserID string, profileID uuid.UUID, limit int, before *time.Time) ([]ChatMessage, *time.Time, error)
}

// ProposalsStorage — интерфейс для хранения AI предложений.
type ProposalsStorage interface {
	// InsertMany сохраняет предложения ассистента и возвращает сохранённые записи.
	InsertMany(ctx context.Context, ownerUserID string, profileID uuid.UUID, drafts []ProposalDraft) ([]AIProposal, error)

	// Get возвращает предложение по id в рамках owner.
	Get(ctx context.Context, ownerUserID string, proposalID uuid.UUID) (AIProposal, bool, error)

	// UpdateStatus обновляет статус предложения в рамках owner.
	UpdateStatus(ctx context.Context, ownerUserID string, proposalID uuid.UUID, status string) error

	// List возвращает предложения по owner/profile с опциональным статусом.
	List(ctx context.Context, ownerUserID string, profileID uuid.UUID, status string, limit int) ([]AIProposal, error)
}

// ChatMessage — сохранённое сообщение чата.
type ChatMessage struct {
	ID          uuid.UUID
	OwnerUserID string
	ProfileID   uuid.UUID
	Role        string
	Content     string
	CreatedAt   time.Time
}

// AIProposal — сохранённое структурированное предложение ассистента.
type AIProposal struct {
	ID          uuid.UUID
	OwnerUserID string
	ProfileID   uuid.UUID
	CreatedAt   time.Time
	Status      string
	Kind        string
	Title       string
	Summary     string
	Payload     []byte
}

// ProposalDraft — draft для сохранения предложений ассистента.
type ProposalDraft struct {
	Kind    string
	Title   string
	Summary string
	Payload []byte
}

// ============================================================================
// Workouts
// ============================================================================

// WorkoutPlansStorage manages workout plans.
type WorkoutPlansStorage interface {
	// GetActivePlan returns the active plan for a profile. Returns false if not found.
	GetActivePlan(ownerUserID string, profileID uuid.UUID) (WorkoutPlan, bool, error)
	// UpsertActivePlan creates or updates the active plan (deactivates old ones).
	UpsertActivePlan(ownerUserID string, profileID uuid.UUID, title string, goal string) (WorkoutPlan, error)
}

// WorkoutPlanItemsStorage manages items within workout plans.
type WorkoutPlanItemsStorage interface {
	// ListItems returns all items for a plan (filtered by ownership).
	ListItems(ownerUserID string, profileID uuid.UUID, planID uuid.UUID) ([]WorkoutPlanItem, error)
	// ReplaceAllItems atomically replaces all items for a plan.
	ReplaceAllItems(ownerUserID string, profileID uuid.UUID, planID uuid.UUID, items []WorkoutItemUpsert) ([]WorkoutPlanItem, error)
	// DeleteItem deletes a single item (ownership check).
	DeleteItem(ownerUserID string, itemID uuid.UUID) error
}

// WorkoutCompletionsStorage manages workout completion records.
type WorkoutCompletionsStorage interface {
	// UpsertCompletion creates or updates a completion record.
	UpsertCompletion(ownerUserID string, profileID uuid.UUID, date string, planItemID uuid.UUID, status string, note string) (WorkoutCompletion, error)
	// ListCompletions returns completions in a date range.
	ListCompletions(ownerUserID string, profileID uuid.UUID, from string, to string) ([]WorkoutCompletion, error)
}

// WorkoutPlan represents a workout plan.
type WorkoutPlan struct {
	ID          uuid.UUID
	OwnerUserID string
	ProfileID   uuid.UUID
	Title       string
	Goal        string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// WorkoutPlanItem represents a single item in a workout plan.
type WorkoutPlanItem struct {
	ID          uuid.UUID
	PlanID      uuid.UUID
	OwnerUserID string
	ProfileID   uuid.UUID
	Kind        string
	TimeMinutes int
	DaysMask    int
	DurationMin int
	Intensity   string
	Note        string
	Details     []byte // JSON
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// WorkoutItemUpsert is used for creating/updating items.
type WorkoutItemUpsert struct {
	Kind        string
	TimeMinutes int
	DaysMask    int
	DurationMin int
	Intensity   string
	Note        string
	Details     []byte
}

// WorkoutCompletion represents a completion/skip record.
type WorkoutCompletion struct {
	ID          uuid.UUID
	OwnerUserID string
	ProfileID   uuid.UUID
	Date        string // YYYY-MM-DD
	PlanItemID  uuid.UUID
	Status      string // done, skipped
	Note        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NutritionTargetsStorage — интерфейс для работы с целями по питанию
type NutritionTargetsStorage interface {
	// Get возвращает цели по питанию для профиля
	Get(ctx context.Context, ownerUserID string, profileID uuid.UUID) (*NutritionTarget, error)

	// Upsert создаёт или обновляет цели по питанию
	Upsert(ctx context.Context, ownerUserID string, profileID uuid.UUID, upsert NutritionTargetUpsert) (*NutritionTarget, error)
}

// NutritionTarget represents nutrition goals/targets for a profile.
type NutritionTarget struct {
	ID           uuid.UUID
	OwnerUserID  string
	ProfileID    uuid.UUID
	CaloriesKcal int
	ProteinG     int
	FatG         int
	CarbsG       int
	CalciumMg    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NutritionTargetUpsert is used for creating/updating targets.
type NutritionTargetUpsert struct {
	CaloriesKcal int
	ProteinG     int
	FatG         int
	CarbsG       int
	CalciumMg    int
}

// FoodPrefsStorage manages user-defined food items with nutritional info
type FoodPrefsStorage interface {
	// List returns food preferences with optional search query
	List(ctx context.Context, ownerUserID string, profileID string, query string, limit, offset int) ([]FoodPref, int, error)
	// Upsert creates or updates a food preference
	Upsert(ctx context.Context, ownerUserID string, profileID string, req FoodPrefUpsert) (FoodPref, error)
	// Delete removes a food preference by ID
	Delete(ctx context.Context, ownerUserID string, id string) error
}

type FoodPref struct {
	ID              string
	OwnerUserID     string
	ProfileID       string
	Name            string
	Tags            []string
	KcalPer100g     int
	ProteinGPer100g int
	FatGPer100g     int
	CarbsGPer100g   int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type FoodPrefUpsert struct {
	ID              string // if empty, create new
	Name            string
	Tags            []string
	KcalPer100g     int
	ProteinGPer100g int
	FatGPer100g     int
	CarbsGPer100g   int
}

// MealPlansStorage manages meal plans (weekly meal schedules)
type MealPlansStorage interface {
	// GetActive returns the active meal plan for a profile
	GetActive(ctx context.Context, ownerUserID string, profileID string) (MealPlan, []MealPlanItem, bool, error)
	// ReplaceActive atomically replaces the active meal plan with new title and items
	ReplaceActive(ctx context.Context, ownerUserID string, profileID string, title string, items []MealPlanItemUpsert) (MealPlan, []MealPlanItem, error)
	// DeleteActive removes the active meal plan
	DeleteActive(ctx context.Context, ownerUserID string, profileID string) error
	// GetToday returns meal items for a specific date (calculates day_index from date)
	GetToday(ctx context.Context, ownerUserID string, profileID string, date time.Time) ([]MealPlanItem, error)
}

type MealPlan struct {
	ID          string
	OwnerUserID string
	ProfileID   string
	Title       string
	IsActive    bool
	FromDate    *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type MealPlanItem struct {
	ID             string
	OwnerUserID    string
	ProfileID      string
	PlanID         string
	DayIndex       int
	MealSlot       string
	Title          string
	Notes          string
	ApproxKcal     int
	ApproxProteinG int
	ApproxFatG     int
	ApproxCarbsG   int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type MealPlanItemUpsert struct {
	DayIndex       int
	MealSlot       string
	Title          string
	Notes          string
	ApproxKcal     int
	ApproxProteinG int
	ApproxFatG     int
	ApproxCarbsG   int
}
