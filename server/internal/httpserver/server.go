package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/fdg312/health-hub/internal/ai"
	"github.com/fdg312/health-hub/internal/auth"
	"github.com/fdg312/health-hub/internal/auth/emailotp"
	"github.com/fdg312/health-hub/internal/blob"
	"github.com/fdg312/health-hub/internal/chat"
	"github.com/fdg312/health-hub/internal/checkins"
	"github.com/fdg312/health-hub/internal/config"
	"github.com/fdg312/health-hub/internal/feed"
	"github.com/fdg312/health-hub/internal/foodprefs"
	"github.com/fdg312/health-hub/internal/intakes"
	"github.com/fdg312/health-hub/internal/mailer"
	"github.com/fdg312/health-hub/internal/mealplans"
	"github.com/fdg312/health-hub/internal/metrics"
	"github.com/fdg312/health-hub/internal/notifications"
	"github.com/fdg312/health-hub/internal/nutrition"
	"github.com/fdg312/health-hub/internal/profiles"
	"github.com/fdg312/health-hub/internal/proposals"
	"github.com/fdg312/health-hub/internal/reports"
	"github.com/fdg312/health-hub/internal/schedules"
	"github.com/fdg312/health-hub/internal/settings"
	"github.com/fdg312/health-hub/internal/sources"
	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/storage/memory"
	"github.com/fdg312/health-hub/internal/storage/postgres"
	"github.com/fdg312/health-hub/internal/workouts"
	"github.com/google/uuid"
)

// Server представляет HTTP сервер
type Server struct {
	config         *config.Config
	mux            *http.ServeMux
	storage        storage.Storage
	authMiddleware *auth.Middleware
}

// New создаёт новый HTTP сервер
func New(cfg *config.Config) *Server {
	s := &Server{
		config: cfg,
		mux:    http.NewServeMux(),
	}

	// Инициализируем storage
	s.initStorage()

	// Регистрируем маршруты
	s.routes()
	return s
}

// initStorage инициализирует storage (Memory или Postgres)
func (s *Server) initStorage() {
	if s.config.DatabaseURL == "" {
		log.Println("Используется in-memory storage")
		s.storage = memory.New()
	} else {
		log.Println("Подключение к PostgreSQL...")
		ctx := context.Background()
		pgStorage, err := postgres.New(ctx, s.config.DatabaseURL)
		if err != nil {
			log.Printf("Ошибка подключения к PostgreSQL: %v", err)
			log.Println("Fallback на in-memory storage")
			s.storage = memory.New()
		} else {
			log.Println("PostgreSQL подключен успешно")
			s.storage = pgStorage
		}
	}
}

// routes регистрирует маршруты
func (s *Server) routes() {
	// Health check (no auth required)
	s.mux.HandleFunc("/healthz", s.handleHealthz)

	// Auth API (no auth required)
	var appleVerifier auth.AppleTokenVerifier
	if s.config.AuthMode == "siwa" {
		appleVerifier = auth.NewRealAppleTokenVerifier(s.config)
	} else {
		appleVerifier = &auth.MockAppleTokenVerifier{}
	}
	authService := auth.NewService(s.config, s.storage, appleVerifier)
	otpStorage := s.getEmailOTPStorage()
	emailSender, err := mailer.NewSenderFromConfig(s.config, log.Default())
	if err != nil {
		if s.config.EmailAuthEnabled {
			log.Fatalf("email sender initialization failed: %v", err)
		}
		log.Printf("email sender initialization skipped (email auth disabled): %v", err)
		emailSender = mailer.NewLocalSender(log.Default())
	}
	emailOTPService := emailotp.NewService(s.config, otpStorage, emailSender)
	authHandler := auth.NewHandlers(authService).WithEmailOTP(emailOTPService)
	s.authMiddleware = auth.NewMiddleware(s.config, authService)

	// POST /v1/auth/dev - local dev token without Apple
	s.mux.HandleFunc("POST /v1/auth/dev", authHandler.HandleDevAuth)

	// POST /v1/auth/siwa - Sign in with Apple (identity token verification via JWKS)
	s.mux.HandleFunc("POST /v1/auth/siwa", authHandler.HandleSignInSIWA)

	// POST /v1/auth/email/request - send OTP to email
	s.mux.HandleFunc("POST /v1/auth/email/request", authHandler.HandleEmailOTPRequest)

	// POST /v1/auth/email/verify - verify OTP and issue JWT
	s.mux.HandleFunc("POST /v1/auth/email/verify", authHandler.HandleEmailOTPVerify)

	// POST /v1/auth/apple - sign in with Apple
	s.mux.HandleFunc("POST /v1/auth/apple", authHandler.HandleSignInApple)

	// Profiles API
	profileService := profiles.NewService(s.storage)
	profileHandler := profiles.NewHandler(profileService)

	// GET /v1/profiles - list all profiles
	s.mux.HandleFunc("GET /v1/profiles", profileHandler.HandleList)

	// POST /v1/profiles - create profile
	s.mux.HandleFunc("POST /v1/profiles", profileHandler.HandleCreate)

	// PATCH /v1/profiles/{id} - update profile
	s.mux.HandleFunc("PATCH /v1/profiles/", profileHandler.HandleUpdate)

	// DELETE /v1/profiles/{id} - delete profile
	s.mux.HandleFunc("DELETE /v1/profiles/", profileHandler.HandleDelete)

	// Metrics API
	// Используем s.storage который реализует и Storage и MetricsStorage
	metricsService := metrics.NewService(s.storage, s.storage.(storage.MetricsStorage))
	metricsHandler := metrics.NewHandler(metricsService)

	// POST /v1/sync/batch - batch sync
	s.mux.HandleFunc("POST /v1/sync/batch", metricsHandler.HandleSyncBatch)

	// GET /v1/metrics/daily - daily metrics
	s.mux.HandleFunc("GET /v1/metrics/daily", metricsHandler.HandleGetDailyMetrics)

	// GET /v1/metrics/hourly - hourly metrics
	s.mux.HandleFunc("GET /v1/metrics/hourly", metricsHandler.HandleGetHourlyMetrics)

	// Checkins API
	checkinsStorage := s.getCheckinsStorage()
	profileAdapter := &profileStorageAdapter{storage: s.storage}
	checkinsService := checkins.NewService(checkinsStorage, profileAdapter)

	// GET /v1/checkins - list checkins
	s.mux.HandleFunc("GET /v1/checkins", checkins.HandleList(checkinsService))

	// POST /v1/checkins - upsert checkin
	s.mux.HandleFunc("POST /v1/checkins", checkins.HandleUpsert(checkinsService))

	// DELETE /v1/checkins/{id} - delete checkin
	s.mux.HandleFunc("DELETE /v1/checkins/{id}", checkins.HandleDelete(checkinsService))

	// Feed API
	metricsStorageAdapter := &metricsStorageAdapter{storage: s.storage.(storage.MetricsStorage)}
	checkinsStorageAdapter := &checkinsStorageAdapter{storage: checkinsStorage}
	intakesStorageAdapter := &intakesStorageAdapter{
		supplementsStorage: s.getSupplementsStorage(),
		intakesStorage:     s.getIntakesStorage(),
	}
	nutritionTargetsStorageAdapter := &nutritionTargetsStorageAdapter{
		storage: s.getNutritionTargetsStorage(),
	}
	mealPlansStorageAdapter := &mealPlansStorageAdapter{
		storage: s.getMealPlansStorage(),
	}
	foodPrefsStorageAdapter := &foodPrefsStorageAdapter{
		storage: s.getFoodPrefsStorage(),
	}
	feedService := feed.NewService(metricsStorageAdapter, checkinsStorageAdapter, profileAdapter, intakesStorageAdapter).
		WithNutritionTargetsStorage(nutritionTargetsStorageAdapter).
		WithMealPlansStorage(mealPlansStorageAdapter).
		WithFoodPrefsStorage(foodPrefsStorageAdapter)

	// GET /v1/feed/day - day summary
	s.mux.HandleFunc("GET /v1/feed/day", feed.HandleGetDay(feedService))

	// User Settings API
	settingsService := settings.NewService(s.getSettingsStorage(), s.config)
	settingsHandler := settings.NewHandler(settingsService)
	s.mux.HandleFunc("GET /v1/settings", settingsHandler.HandleGet)
	s.mux.HandleFunc("PUT /v1/settings", settingsHandler.HandlePut)

	// Chat API
	aiProvider := ai.NewProvider(s.config)
	chatService := chat.NewService(
		s.getChatStorage(),
		s.getProposalsStorage(),
		s.storage,
		feedService,
		settingsService,
		aiProvider,
	)
	chatHandler := chat.NewHandler(chatService)
	s.mux.HandleFunc("GET /v1/chat/messages", chatHandler.HandleListMessages)
	s.mux.HandleFunc("POST /v1/chat/messages", chatHandler.HandleSendMessage)

	// Reports API
	reportsStorage := s.getReportsStorage()
	sourcesBlobStore, reportsBlobStore := s.initBlobStores()
	reportsCheckinsAdapter := &reportsCheckinsAdapter{storage: checkinsStorage}
	reportsProfileAdapter := &reportsProfileAdapter{storage: s.storage}
	reportsService := reports.NewService(
		reportsStorage,
		s.storage.(storage.MetricsStorage),
		reportsCheckinsAdapter,
		reportsProfileAdapter,
		reportsBlobStore,
		s.config.ReportsMaxRangeDays,
		s.config.Blob.S3.PresignTTLSeconds,
		s.config.Blob.S3.PublicBaseURL,
		s.config.Blob.S3.PreferPublicURL,
	)
	reportsHandler := reports.NewHandlers(reportsService)

	// POST /v1/reports - create report
	s.mux.HandleFunc("POST /v1/reports", reportsHandler.HandleCreate)

	// GET /v1/reports - list reports
	s.mux.HandleFunc("GET /v1/reports", reportsHandler.HandleList)

	// GET /v1/reports/{id}/download - download report
	s.mux.HandleFunc("GET /v1/reports/{id}/download", reportsHandler.HandleDownload)

	// DELETE /v1/reports/{id} - delete report
	s.mux.HandleFunc("DELETE /v1/reports/{id}", reportsHandler.HandleDelete)

	// Sources API
	sourcesStorage := s.getSourcesStorage()
	sourcesProfileAdapter := &sourcesProfileAdapter{storage: s.storage}
	sourcesService := sources.NewService(
		sourcesStorage,
		sourcesProfileAdapter,
		sourcesBlobStore,
		s.config.UploadMaxMB,
		s.config.UploadAllowedMime,
		s.config.SourcesMaxPerCheckin,
		s.config.Blob.S3.PublicBaseURL,
		s.config.Blob.S3.PreferPublicURL,
	)
	sourcesHandler := sources.NewHandlers(sourcesService)

	// POST /v1/sources - create link/note source
	s.mux.HandleFunc("POST /v1/sources", sourcesHandler.HandleCreate)

	// POST /v1/sources/image - upload image source
	s.mux.HandleFunc("POST /v1/sources/image", sourcesHandler.HandleCreateImage)

	// GET /v1/sources - list sources
	s.mux.HandleFunc("GET /v1/sources", sourcesHandler.HandleList)

	// GET /v1/sources/{id}/download - download image
	s.mux.HandleFunc("GET /v1/sources/{id}/download", sourcesHandler.HandleDownload)

	// DELETE /v1/sources/{id} - delete source
	s.mux.HandleFunc("DELETE /v1/sources/{id}", sourcesHandler.HandleDelete)

	// Notifications/Inbox API
	notificationsStorage := s.getNotificationsStorage()
	notificationsService := notifications.NewService(
		notificationsStorage,
		s.storage.(storage.MetricsStorage),
		checkinsStorage,
		s.storage,
		s.getSettingsStorage(),
		s.config,
	).WithWorkoutStorages(
		s.getWorkoutPlansStorage(),
		s.getWorkoutPlanItemsStorage(),
		s.getWorkoutCompletionsStorage(),
	).WithMealPlansStorage(
		s.getMealPlansStorage(),
	)
	notificationsHandler := notifications.NewHandler(notificationsService)

	// GET /v1/inbox - list notifications
	s.mux.HandleFunc("GET /v1/inbox", notificationsHandler.HandleList)

	// GET /v1/inbox/unread-count - get unread count
	s.mux.HandleFunc("GET /v1/inbox/unread-count", notificationsHandler.HandleUnreadCount)

	// POST /v1/inbox/mark-read - mark specific notifications as read
	s.mux.HandleFunc("POST /v1/inbox/mark-read", notificationsHandler.HandleMarkRead)

	// POST /v1/inbox/mark-all-read - mark all notifications as read
	s.mux.HandleFunc("POST /v1/inbox/mark-all-read", notificationsHandler.HandleMarkAllRead)

	// POST /v1/inbox/generate - generate notifications for a date
	s.mux.HandleFunc("POST /v1/inbox/generate", notificationsHandler.HandleGenerate)

	// Intakes API (Water & Supplements)
	supplementsStorage := s.getSupplementsStorage()
	intakesStorage := s.getIntakesStorage()
	supplementSchedulesStorage := s.getSupplementSchedulesStorage()
	intakesService := intakes.NewService(
		supplementsStorage,
		intakesStorage,
		s.storage,
		s.config,
	)
	intakesHandler := intakes.NewHandlers(intakesService)

	// Supplement schedules API
	schedulesService := schedules.NewService(
		supplementSchedulesStorage,
		supplementsStorage,
		s.storage,
	)
	schedulesHandler := schedules.NewHandlers(schedulesService)

	// POST /v1/supplements - create supplement
	s.mux.HandleFunc("POST /v1/supplements", intakesHandler.HandleCreateSupplement)

	// GET /v1/supplements - list supplements
	s.mux.HandleFunc("GET /v1/supplements", intakesHandler.HandleListSupplements)

	// PATCH /v1/supplements/{id} - update supplement
	s.mux.HandleFunc("PATCH /v1/supplements/", intakesHandler.HandleUpdateSupplement)

	// DELETE /v1/supplements/{id} - delete supplement
	s.mux.HandleFunc("DELETE /v1/supplements/", intakesHandler.HandleDeleteSupplement)

	// POST /v1/intakes/water - add water intake
	s.mux.HandleFunc("POST /v1/intakes/water", intakesHandler.HandleAddWater)

	// GET /v1/intakes/daily - get daily intakes
	s.mux.HandleFunc("GET /v1/intakes/daily", intakesHandler.HandleGetIntakesDaily)

	// POST /v1/intakes/supplements - upsert supplement intake
	s.mux.HandleFunc("POST /v1/intakes/supplements", intakesHandler.HandleUpsertSupplementIntake)

	// GET /v1/schedules/supplements - list supplement schedules
	s.mux.HandleFunc("GET /v1/schedules/supplements", schedulesHandler.HandleList)

	// POST /v1/schedules/supplements - upsert supplement schedule
	s.mux.HandleFunc("POST /v1/schedules/supplements", schedulesHandler.HandleUpsert)

	// PUT /v1/schedules/supplements/replace - replace schedule set
	s.mux.HandleFunc("PUT /v1/schedules/supplements/replace", schedulesHandler.HandleReplaceAll)

	// DELETE /v1/schedules/supplements/{id} - delete schedule
	s.mux.HandleFunc("DELETE /v1/schedules/supplements/{id}", schedulesHandler.HandleDelete)

	// Workouts API
	workoutPlansStorage := s.getWorkoutPlansStorage()
	workoutItemsStorage := s.getWorkoutPlanItemsStorage()
	workoutCompletionsStorage := s.getWorkoutCompletionsStorage()
	// feedService not needed for MVP - actual workouts extraction can be added later
	workoutsService := workouts.NewService(
		workoutPlansStorage,
		workoutItemsStorage,
		workoutCompletionsStorage,
		s.storage,
		nil, // feedService - actual workouts not needed for MVP
	)
	workoutsHandler := workouts.NewHandlers(workoutsService)

	// GET /v1/workouts/plan - get active workout plan
	s.mux.HandleFunc("GET /v1/workouts/plan", workoutsHandler.HandleGetPlan)

	// PUT /v1/workouts/plan/replace - replace workout plan
	s.mux.HandleFunc("PUT /v1/workouts/plan/replace", workoutsHandler.HandleReplacePlan)

	// POST /v1/workouts/completions - upsert workout completion
	s.mux.HandleFunc("POST /v1/workouts/completions", workoutsHandler.HandleUpsertCompletion)

	// GET /v1/workouts/today - get today's workout plan and status
	s.mux.HandleFunc("GET /v1/workouts/today", workoutsHandler.HandleGetToday)

	// GET /v1/workouts/completions - list completions
	s.mux.HandleFunc("GET /v1/workouts/completions", workoutsHandler.HandleListCompletions)

	// Nutrition Targets API
	nutritionTargetsStorage := s.getNutritionTargetsStorage()
	nutritionService := nutrition.NewService(s.storage, nutritionTargetsStorage)
	nutritionHandler := nutrition.NewHandler(nutritionService)

	// GET /v1/nutrition/targets - get nutrition targets or defaults
	s.mux.HandleFunc("GET /v1/nutrition/targets", nutritionHandler.HandleGetTargets)

	// PUT /v1/nutrition/targets - upsert nutrition targets
	s.mux.HandleFunc("PUT /v1/nutrition/targets", nutritionHandler.HandleUpsertTargets)

	// Food Preferences API
	foodPrefsStorage := s.getFoodPrefsStorage()
	foodPrefsService := foodprefs.NewService(foodPrefsStorage)
	foodPrefsHandler := foodprefs.NewHandler(foodPrefsService)

	// GET /v1/food/prefs - list food preferences
	s.mux.HandleFunc("GET /v1/food/prefs", foodPrefsHandler.HandleList)

	// POST /v1/food/prefs - upsert food preference
	s.mux.HandleFunc("POST /v1/food/prefs", foodPrefsHandler.HandleUpsert)

	// DELETE /v1/food/prefs/{id} - delete food preference
	s.mux.HandleFunc("DELETE /v1/food/prefs/{id}", foodPrefsHandler.HandleDelete)

	// Meal Plans API
	mealPlansStorage := s.getMealPlansStorage()
	mealPlansService := mealplans.NewService(mealPlansStorage)
	mealPlansHandler := mealplans.NewHandler(mealPlansService)

	// GET /v1/meal/plan - get active meal plan
	s.mux.HandleFunc("GET /v1/meal/plan", mealPlansHandler.HandleGet)

	// PUT /v1/meal/plan/replace - replace meal plan
	s.mux.HandleFunc("PUT /v1/meal/plan/replace", mealPlansHandler.HandleReplace)

	// GET /v1/meal/today - get today's meal plan
	s.mux.HandleFunc("GET /v1/meal/today", mealPlansHandler.HandleGetToday)

	// DELETE /v1/meal/plan - delete active meal plan
	s.mux.HandleFunc("DELETE /v1/meal/plan", mealPlansHandler.HandleDelete)

	// AI Proposals API (after workouts and nutrition to allow all proposal kinds)
	proposalsService := proposals.NewService(
		s.getProposalsStorage(),
		s.storage,
		settingsService,
	).WithWorkoutService(workoutsService).WithNutritionService(nutritionService).WithMealPlanService(mealPlansService)
	proposalsHandler := proposals.NewHandler(proposalsService)
	s.mux.HandleFunc("GET /v1/ai/proposals", proposalsHandler.HandleList)
	s.mux.HandleFunc("POST /v1/ai/proposals/{id}/apply", proposalsHandler.HandleApply)
	s.mux.HandleFunc("POST /v1/ai/proposals/{id}/reject", proposalsHandler.HandleReject)
}

// getCheckinsStorage returns the checkins storage based on storage type
func (s *Server) getCheckinsStorage() checkins.Storage {
	switch st := s.storage.(type) {
	case *memory.MemoryStorage:
		return st.GetCheckinsStorage()
	case *postgres.PostgresStorage:
		return st.GetCheckinsStorage()
	default:
		log.Fatal("unknown storage type")
		return nil
	}
}

// getReportsStorage returns the reports storage based on storage type
func (s *Server) getReportsStorage() storage.ReportsStorage {
	switch st := s.storage.(type) {
	case *memory.MemoryStorage:
		return st.GetReportsStorage()
	case *postgres.PostgresStorage:
		return st.GetReportsStorage()
	default:
		log.Fatal("unknown storage type")
		return nil
	}
}

// getSourcesStorage returns the sources storage based on storage type
func (s *Server) getSourcesStorage() storage.SourcesStorage {
	switch st := s.storage.(type) {
	case *memory.MemoryStorage:
		return st.GetSourcesStorage()
	case *postgres.PostgresStorage:
		return st.GetSourcesStorage()
	default:
		log.Fatal("unknown storage type")
		return nil
	}
}

// getNotificationsStorage returns the notifications storage based on storage type
func (s *Server) getNotificationsStorage() storage.NotificationsStorage {
	switch st := s.storage.(type) {
	case *memory.MemoryStorage:
		return st.GetNotificationsStorage()
	case *postgres.PostgresStorage:
		return st.GetNotificationsStorage()
	default:
		log.Fatal("unknown storage type")
		return nil
	}
}

// getSupplementsStorage returns the supplements storage based on storage type
func (s *Server) getSupplementsStorage() storage.SupplementsStorage {
	switch st := s.storage.(type) {
	case *memory.MemoryStorage:
		return st.GetSupplementsStorage()
	case *postgres.PostgresStorage:
		return st.GetSupplementsStorage()
	default:
		log.Fatal("unknown storage type")
		return nil
	}
}

// getIntakesStorage returns the intakes storage based on storage type
func (s *Server) getIntakesStorage() storage.IntakesStorage {
	switch st := s.storage.(type) {
	case *memory.MemoryStorage:
		return st.GetIntakesStorage()
	case *postgres.PostgresStorage:
		return st.GetIntakesStorage()
	default:
		log.Fatal("unknown storage type")
		return nil
	}
}

// getSupplementSchedulesStorage returns supplement schedules storage based on storage type.
func (s *Server) getSupplementSchedulesStorage() storage.SupplementSchedulesStorage {
	switch st := s.storage.(type) {
	case *memory.MemoryStorage:
		return st.GetSupplementSchedulesStorage()
	case *postgres.PostgresStorage:
		return st.GetSupplementSchedulesStorage()
	default:
		log.Fatal("unknown storage type")
		return nil
	}
}

// getEmailOTPStorage returns the email OTP storage based on storage type.
func (s *Server) getEmailOTPStorage() storage.EmailOTPStorage {
	switch st := s.storage.(type) {
	case *memory.MemoryStorage:
		return st.GetEmailOTPStorage()
	case *postgres.PostgresStorage:
		return st.GetEmailOTPStorage()
	default:
		log.Fatal("unknown storage type")
		return nil
	}
}

// getSettingsStorage returns the user settings storage based on storage type.
func (s *Server) getSettingsStorage() storage.SettingsStorage {
	switch st := s.storage.(type) {
	case *memory.MemoryStorage:
		return st.GetSettingsStorage()
	case *postgres.PostgresStorage:
		return st.GetSettingsStorage()
	default:
		log.Fatal("unknown storage type")
		return nil
	}
}

// getChatStorage returns chat storage based on storage type.
func (s *Server) getChatStorage() storage.ChatStorage {
	switch st := s.storage.(type) {
	case *memory.MemoryStorage:
		return st.GetChatStorage()
	case *postgres.PostgresStorage:
		return st.GetChatStorage()
	default:
		log.Fatal("unknown storage type")
		return nil
	}
}

// getWorkoutPlansStorage returns workout plans storage based on storage type.
func (s *Server) getWorkoutPlansStorage() storage.WorkoutPlansStorage {
	switch st := s.storage.(type) {
	case *memory.MemoryStorage:
		return st.GetWorkoutPlansStorage()
	case *postgres.PostgresStorage:
		return st.GetWorkoutPlansStorage()
	default:
		log.Fatal("unknown storage type")
		return nil
	}
}

// getWorkoutPlanItemsStorage returns workout plan items storage based on storage type.
func (s *Server) getWorkoutPlanItemsStorage() storage.WorkoutPlanItemsStorage {
	switch st := s.storage.(type) {
	case *memory.MemoryStorage:
		return st.GetWorkoutPlanItemsStorage()
	case *postgres.PostgresStorage:
		return st.GetWorkoutPlanItemsStorage()
	default:
		log.Fatal("unknown storage type")
		return nil
	}
}

// getWorkoutCompletionsStorage returns workout completions storage based on storage type.
func (s *Server) getWorkoutCompletionsStorage() storage.WorkoutCompletionsStorage {
	switch st := s.storage.(type) {
	case *memory.MemoryStorage:
		return st.GetWorkoutCompletionsStorage()
	case *postgres.PostgresStorage:
		return st.GetWorkoutCompletionsStorage()
	default:
		panic("unsupported storage type")
	}
}

func (s *Server) getNutritionTargetsStorage() storage.NutritionTargetsStorage {
	switch st := s.storage.(type) {
	case *memory.MemoryStorage:
		return st.GetNutritionTargetsStorage()
	case *postgres.PostgresStorage:
		return st.GetNutritionTargetsStorage()
	default:
		panic("unsupported storage type")
	}
}

// getFoodPrefsStorage returns food prefs storage based on storage type.
func (s *Server) getFoodPrefsStorage() storage.FoodPrefsStorage {
	switch st := s.storage.(type) {
	case *memory.MemoryStorage:
		return st.GetFoodPrefsStorage()
	case *postgres.PostgresStorage:
		return st.GetFoodPrefsStorage()
	default:
		panic("unsupported storage type")
	}
}

// getMealPlansStorage returns meal plans storage based on storage type.
func (s *Server) getMealPlansStorage() storage.MealPlansStorage {
	switch st := s.storage.(type) {
	case *memory.MemoryStorage:
		return st.GetMealPlansStorage()
	case *postgres.PostgresStorage:
		return st.GetMealPlansStorage()
	default:
		panic("unsupported storage type")
	}
}

// getProposalsStorage returns proposals storage based on storage type.
func (s *Server) getProposalsStorage() storage.ProposalsStorage {
	switch st := s.storage.(type) {
	case *memory.MemoryStorage:
		return st.GetProposalsStorage()
	case *postgres.PostgresStorage:
		return st.GetProposalsStorage()
	default:
		log.Fatal("unknown storage type")
		return nil
	}
}

// initBlobStores initializes blob stores for sources and reports.
// Sources always follow BLOB_MODE, reports may override via REPORTS_MODE.
func (s *Server) initBlobStores() (sourcesStore blob.Store, reportsStore blob.Store) {
	// Initialize sources blob store
	sourcesCfg := s.config.Blob
	sourcesCfg.ReportsModeSet = false
	sourcesCfg.ReportsMode = sourcesCfg.Mode

	log.Printf("INFO blob: initializing sources store (BLOB_MODE=%s)", sourcesCfg.Mode)
	baseStore, baseMode, err := blob.NewBlobStore(sourcesCfg, log.Default())
	if err != nil {
		log.Fatalf("FATAL blob: failed to initialize sources store: %v", err)
	}
	log.Printf("INFO blob: sources blob mode: %s", baseMode)

	// Initialize reports blob store (may override)
	effectiveReportsMode := s.config.Blob.EffectiveReportsMode()
	if !s.config.Blob.ReportsModeSet || effectiveReportsMode == s.config.Blob.Mode {
		log.Printf("INFO blob: reports blob mode: %s (same as sources)", baseMode)
		return baseStore, baseStore
	}

	log.Printf("INFO blob: initializing reports store (REPORTS_MODE=%s, override from BLOB_MODE=%s)", effectiveReportsMode, s.config.Blob.Mode)
	reportsCfg := s.config.Blob
	reportsCfg.Mode = effectiveReportsMode
	reportsCfg.ReportsModeSet = false
	reportsCfg.ReportsMode = effectiveReportsMode

	reportsBlobStore, reportsMode, err := blob.NewBlobStore(reportsCfg, log.Default())
	if err != nil {
		log.Fatalf("FATAL blob: failed to initialize reports store: %v", err)
	}

	// If override resolves to same mode, reuse the base store/client.
	if reportsMode == baseMode {
		log.Printf("INFO blob: reports blob mode: %s (resolved to same as sources, reusing store)", reportsMode)
		return baseStore, baseStore
	}

	log.Printf("INFO blob: reports blob mode: %s (separate store)", reportsMode)
	return baseStore, reportsBlobStore
}

// profileStorageAdapter adapts storage.Storage to checkins.ProfileStorage
type profileStorageAdapter struct {
	storage storage.Storage
}

func (p *profileStorageAdapter) GetProfile(ctx context.Context, id uuid.UUID) (*storage.Profile, error) {
	return p.storage.GetProfile(ctx, id)
}

// metricsStorageAdapter adapts storage.MetricsStorage to feed.MetricsStorage
type metricsStorageAdapter struct {
	storage storage.MetricsStorage
}

func (m *metricsStorageAdapter) GetDailyMetrics(ctx context.Context, profileID uuid.UUID, from, to string) ([]feed.DailyMetricRow, error) {
	rows, err := m.storage.GetDailyMetrics(ctx, profileID, from, to)
	if err != nil {
		return nil, err
	}

	result := make([]feed.DailyMetricRow, len(rows))
	for i, row := range rows {
		result[i] = feed.DailyMetricRow{
			ProfileID: row.ProfileID,
			Date:      row.Date,
			Payload:   row.Payload,
		}
	}
	return result, nil
}

// checkinsStorageAdapter adapts checkins.Storage to feed.CheckinsStorage
type checkinsStorageAdapter struct {
	storage checkins.Storage
}

func (c *checkinsStorageAdapter) ListCheckins(ctx context.Context, profileID uuid.UUID, from, to string) ([]feed.Checkin, error) {
	checkinsRows, err := c.storage.ListCheckins(profileID, from, to)
	if err != nil {
		return nil, err
	}

	result := make([]feed.Checkin, len(checkinsRows))
	for i, row := range checkinsRows {
		result[i] = feed.Checkin{
			ID:        row.ID,
			ProfileID: row.ProfileID,
			Date:      row.Date,
			Type:      row.Type,
			Score:     row.Score,
			Tags:      row.Tags,
			Note:      row.Note,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
	}
	return result, nil
}

// reportsCheckinsAdapter adapts checkins.Storage to reports.CheckinsStorageAdapter
type reportsCheckinsAdapter struct {
	storage checkins.Storage
}

func (r *reportsCheckinsAdapter) ListCheckins(ctx context.Context, profileID uuid.UUID, from, to string) ([]reports.Checkin, error) {
	checkinsRows, err := r.storage.ListCheckins(profileID, from, to)
	if err != nil {
		return nil, err
	}

	result := make([]reports.Checkin, len(checkinsRows))
	for i, row := range checkinsRows {
		result[i] = reports.Checkin{
			ID:        row.ID,
			ProfileID: row.ProfileID,
			Date:      row.Date,
			Type:      row.Type,
			Score:     row.Score,
			Tags:      row.Tags,
			Note:      row.Note,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
	}
	return result, nil
}

// reportsProfileAdapter adapts storage.Storage to reports.ProfileStorageAdapter
type reportsProfileAdapter struct {
	storage storage.Storage
}

func (r *reportsProfileAdapter) GetProfile(ctx context.Context, id uuid.UUID) (*storage.Profile, error) {
	return r.storage.GetProfile(ctx, id)
}

// sourcesProfileAdapter adapts storage.Storage to sources.ProfileStorageAdapter
type sourcesProfileAdapter struct {
	storage storage.Storage
}

func (s *sourcesProfileAdapter) GetProfile(ctx context.Context, id uuid.UUID) (*storage.Profile, error) {
	return s.storage.GetProfile(ctx, id)
}

// intakesStorageAdapter adapts intakes storage to feed.IntakesStorage
type intakesStorageAdapter struct {
	supplementsStorage storage.SupplementsStorage
	intakesStorage     storage.IntakesStorage
}

func (i *intakesStorageAdapter) GetWaterDaily(ctx context.Context, profileID uuid.UUID, date string) (int, error) {
	return i.intakesStorage.GetWaterDaily(ctx, profileID, date)
}

func (i *intakesStorageAdapter) GetSupplementDaily(ctx context.Context, profileID uuid.UUID, date string) (map[uuid.UUID]string, error) {
	return i.intakesStorage.GetSupplementDaily(ctx, profileID, date)
}

func (i *intakesStorageAdapter) ListSupplements(ctx context.Context, profileID uuid.UUID) ([]feed.Supplement, error) {
	supplements, err := i.supplementsStorage.ListSupplements(ctx, profileID)
	if err != nil {
		return nil, err
	}

	result := make([]feed.Supplement, len(supplements))
	for i, sup := range supplements {
		result[i] = feed.Supplement{ID: sup.ID}
	}
	return result, nil
}

// nutritionTargetsStorageAdapter adapts storage.NutritionTargetsStorage to feed.NutritionTargetsStorage
type nutritionTargetsStorageAdapter struct {
	storage storage.NutritionTargetsStorage
}

func (n *nutritionTargetsStorageAdapter) Get(ctx context.Context, ownerUserID string, profileID uuid.UUID) (*feed.NutritionTarget, error) {
	target, err := n.storage.Get(ctx, ownerUserID, profileID)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, nil
	}
	return &feed.NutritionTarget{
		CaloriesKcal: target.CaloriesKcal,
		ProteinG:     target.ProteinG,
		FatG:         target.FatG,
		CarbsG:       target.CarbsG,
		CalciumMg:    target.CalciumMg,
	}, nil
}

// mealPlansStorageAdapter adapts storage.MealPlansStorage to feed.MealPlansStorage
type mealPlansStorageAdapter struct {
	storage storage.MealPlansStorage
}

func (m *mealPlansStorageAdapter) GetToday(ctx context.Context, ownerUserID string, profileID string, date time.Time) ([]feed.MealPlanItemStorage, error) {
	items, err := m.storage.GetToday(ctx, ownerUserID, profileID, date)
	if err != nil {
		return nil, err
	}

	result := make([]feed.MealPlanItemStorage, len(items))
	for i, item := range items {
		result[i] = feed.MealPlanItemStorage{
			ID:             item.ID,
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
	return result, nil
}

func (m *mealPlansStorageAdapter) GetActive(ctx context.Context, ownerUserID string, profileID string) (feed.MealPlanStorage, []feed.MealPlanItemStorage, bool, error) {
	plan, items, found, err := m.storage.GetActive(ctx, ownerUserID, profileID)
	if err != nil || !found {
		return feed.MealPlanStorage{}, nil, found, err
	}

	planResult := feed.MealPlanStorage{
		ID:    plan.ID,
		Title: plan.Title,
	}

	itemsResult := make([]feed.MealPlanItemStorage, len(items))
	for i, item := range items {
		itemsResult[i] = feed.MealPlanItemStorage{
			ID:             item.ID,
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
	return planResult, itemsResult, true, nil
}

// foodPrefsStorageAdapter adapts storage.FoodPrefsStorage to feed.FoodPrefsStorage
type foodPrefsStorageAdapter struct {
	storage storage.FoodPrefsStorage
}

func (f *foodPrefsStorageAdapter) List(ctx context.Context, ownerUserID string, profileID string, query string, limit, offset int) ([]feed.FoodPref, int, error) {
	prefs, total, err := f.storage.List(ctx, ownerUserID, profileID, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	result := make([]feed.FoodPref, len(prefs))
	for i, pref := range prefs {
		result[i] = feed.FoodPref{
			ID: pref.ID,
		}
	}
	return result, total, nil
}

// handleHealthz возвращает статус сервера
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// Start запускает HTTP сервер
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Port)

	// Build middleware chain (outermost first): CORS → Rate Limit → Auth → Router
	var handler http.Handler = s.mux
	if s.authMiddleware != nil && s.config.AuthMode != "none" {
		if s.config.AuthRequired {
			handler = s.authMiddleware.RequireAuth(handler)
		} else {
			handler = s.authMiddleware.OptionalAuth(handler)
		}
	}
	handler = RateLimitMiddleware(s.config, handler)
	handler = CORSMiddleware(s.config, handler)

	log.Printf("Сервер запущен на http://localhost%s\n", addr)
	log.Printf("Health check: http://localhost%s/healthz\n", addr)
	log.Printf("Profiles API: http://localhost%s/v1/profiles\n", addr)

	return http.ListenAndServe(addr, handler)
}

// Close закрывает storage и освобождает ресурсы
func (s *Server) Close() error {
	if s.storage != nil {
		return s.storage.Close()
	}
	return nil
}
