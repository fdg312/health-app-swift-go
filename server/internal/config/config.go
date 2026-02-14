package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	BlobModeLocal = "local"
	BlobModeS3    = "s3"
	BlobModeAuto  = "auto"
)

type S3Config struct {
	Endpoint          string
	Region            string
	Bucket            string
	AccessKeyID       string
	SecretAccessKey   string
	PublicBaseURL     string
	PresignTTLSeconds int
	PreferPublicURL   bool
}

func (c S3Config) MissingRequired() []string {
	missing := make([]string, 0, 6)
	if strings.TrimSpace(c.Endpoint) == "" {
		missing = append(missing, "S3_ENDPOINT")
	}
	if strings.TrimSpace(c.Region) == "" {
		missing = append(missing, "S3_REGION")
	}
	if strings.TrimSpace(c.Bucket) == "" {
		missing = append(missing, "S3_BUCKET")
	}
	if strings.TrimSpace(c.AccessKeyID) == "" {
		missing = append(missing, "S3_ACCESS_KEY_ID")
	}
	if strings.TrimSpace(c.SecretAccessKey) == "" {
		missing = append(missing, "S3_SECRET_ACCESS_KEY")
	}
	if strings.TrimSpace(c.PublicBaseURL) == "" {
		missing = append(missing, "S3_PUBLIC_BASE_URL")
	}
	return missing
}

func (c S3Config) IsConfigured() bool {
	return len(c.MissingRequired()) == 0
}

func (c S3Config) Diagnostics() (level string, code string, msg string) {
	allEmpty := strings.TrimSpace(c.Endpoint) == "" &&
		strings.TrimSpace(c.Region) == "" &&
		strings.TrimSpace(c.Bucket) == "" &&
		strings.TrimSpace(c.AccessKeyID) == "" &&
		strings.TrimSpace(c.SecretAccessKey) == "" &&
		strings.TrimSpace(c.PublicBaseURL) == ""

	if allEmpty {
		return "INFO", "s3_not_configured", "not configured (all empty)"
	}

	missing := c.MissingRequired()
	if len(missing) > 0 {
		return "WARN", "s3_partial_config", fmt.Sprintf("partial config, missing=%v", missing)
	}

	return "INFO", "s3_ready", "ready"
}

// DiagnosticsSummary returns a detailed summary for logging (no secrets)
func (c S3Config) DiagnosticsSummary() string {
	accessKeyStatus := "not set"
	if strings.TrimSpace(c.AccessKeyID) != "" {
		accessKeyStatus = "set"
	}
	secretKeyStatus := "not set"
	if strings.TrimSpace(c.SecretAccessKey) != "" {
		secretKeyStatus = "set"
	}

	return fmt.Sprintf("endpoint=%s region=%s bucket=%s public_base_url=%s presign_ttl=%ds prefer_public_url=%t access_key_id=%s secret_access_key=%s",
		nonEmptyOrDash(c.Endpoint),
		nonEmptyOrDash(c.Region),
		nonEmptyOrDash(c.Bucket),
		nonEmptyOrDash(c.PublicBaseURL),
		c.PresignTTLSeconds,
		c.PreferPublicURL,
		accessKeyStatus,
		secretKeyStatus,
	)
}

func nonEmptyOrDash(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "-"
	}
	return v
}

type BlobConfig struct {
	Mode           string // local|s3|auto
	ReportsMode    string // local|s3|auto (override)
	ReportsModeSet bool
	S3             S3Config
}

func (c BlobConfig) EffectiveReportsMode() string {
	if c.ReportsModeSet {
		return c.ReportsMode
	}
	return c.Mode
}

// Config содержит конфигурацию приложения
type Config struct {
	Env      string // local | staging | prod
	Port     int
	LogLevel string

	// Database
	DatabaseURL       string // runtime connection (resolved: pooled > url > direct)
	DatabaseURLRaw    string // DATABASE_URL as provided
	DatabaseURLPooled string // DATABASE_URL_POOLED as provided
	DatabaseURLDirect string // for migrations / DDL (may be empty)

	// CORS
	CORSAllowedOrigins   []string
	CORSAllowCredentials bool

	// Rate Limiting
	RateLimitRPS   int
	RateLimitBurst int

	// S3 (Yandex Object Storage)
	// Deprecated flat fields kept for backward compatibility; use Blob.S3.
	S3Endpoint        string
	S3Region          string
	S3Bucket          string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3PublicBaseURL   string
	S3PresignTTL      int
	Blob              BlobConfig

	// Reports
	ReportsMaxRangeDays    int
	ReportsDefaultTTLHours int

	// Uploads / Sources
	UploadMaxMB          int
	UploadAllowedMime    string
	SourcesMaxPerCheckin int

	// Inbox / Notifications
	NotificationsMaxPerDay     int
	DefaultSleepMinMinutes     int
	DefaultStepsMin            int
	DefaultActiveEnergyMinKcal int

	// Intakes (Water & Supplements)
	IntakesMaxWaterMlPerDay  int
	IntakesWaterDefaultAddMl int
	IntakesMaxSupplements    int

	// Authentication & Authorization
	AuthMode            string // none | dev | siwa
	AuthEnabled         bool   // backward-compatible derived flag
	AuthRequired        bool
	EmailAuthEnabled    bool
	JWTSecret           string
	OTPSecret           string
	JWTIssuer           string
	JWTTTLMinutes       int
	OTPTTLSeconds       int
	OTPMaxAttempts      int
	OTPResendMinSeconds int
	OTPMaxSendPerHour   int
	AppleBundleID       string
	AppleIssuer         string
	AppleJWKSURL        string
	AppleSubPrefix      string
	EmailSenderMode     string // local | smtp | resend
	SMTPHost            string
	SMTPPort            int
	SMTPUsername        string
	SMTPPassword        string
	SMTPFrom            string
	SMTPUseTLS          bool
	ResendAPIKey        string
	ResendFrom          string
	OTPDebugReturnCode  bool

	// AI
	AIMode            string // mock | openai
	AIMaxOutputTokens int
	AITemperature     float64
	AITimeoutSeconds  int
	OpenAIAPIKey      string
	OpenAIModel       string

	// Migrations
	RunMigrationsOnStartup bool
}

// Load загружает конфигурацию из переменных окружения
func Load() *Config {
	// APP_ENV (fallback to ENV for backward compat, default: local)
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = os.Getenv("ENV")
	}
	if env == "" {
		env = "local"
	}

	// PORT (default: 8080)
	port := 8080
	if portStr := os.Getenv("PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	// LOG_LEVEL (default: debug)
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "debug"
	}

	// ---------- Database ----------
	// Priority: DATABASE_URL_POOLED > DATABASE_URL > DATABASE_URL_DIRECT
	dbPooled := strings.TrimSpace(os.Getenv("DATABASE_URL_POOLED"))
	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	dbDirect := strings.TrimSpace(os.Getenv("DATABASE_URL_DIRECT"))

	runtimeDB := dbPooled
	if runtimeDB == "" {
		runtimeDB = dbURL
	}
	if runtimeDB == "" {
		runtimeDB = dbDirect
	}

	// ---------- Migrations ----------
	runMigrationsOnStartup := parseBoolEnv("RUN_MIGRATIONS_ON_STARTUP")

	// ---------- CORS ----------
	corsOrigins := parseCORSOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"), env)
	corsAllowCreds := os.Getenv("CORS_ALLOW_CREDENTIALS") == "1"

	// ---------- Rate Limiting ----------
	rateLimitRPS := envInt("RATE_LIMIT_RPS", 0)
	rateLimitBurst := envInt("RATE_LIMIT_BURST", 0)

	// ---------- Blob / S3 ----------
	blobMode := parseBlobMode("BLOB_MODE", BlobModeLocal)
	reportsModeRaw := strings.ToLower(strings.TrimSpace(os.Getenv("REPORTS_MODE")))
	reportsModeSet := reportsModeRaw != ""
	reportsMode := reportsModeRaw
	if reportsMode == "" {
		reportsMode = BlobModeLocal
	}
	if reportsMode != BlobModeLocal && reportsMode != BlobModeS3 && reportsMode != BlobModeAuto {
		log.Printf("WARNING: unknown REPORTS_MODE=%q, fallback to %s", reportsMode, BlobModeLocal)
		reportsMode = BlobModeLocal
	}

	// S3_PRESIGN_TTL_SECONDS (default: 900, enforce > 0)
	s3PresignTTL := envInt("S3_PRESIGN_TTL_SECONDS", 900)
	if s3PresignTTL <= 0 {
		s3PresignTTL = 900
	}

	// S3_PREFER_PUBLIC_URL (default: 0)
	s3PreferPublicURL := parseBoolEnv("S3_PREFER_PUBLIC_URL")

	s3Cfg := S3Config{
		Endpoint:          strings.TrimSpace(os.Getenv("S3_ENDPOINT")),
		Region:            strings.TrimSpace(os.Getenv("S3_REGION")),
		Bucket:            strings.TrimSpace(os.Getenv("S3_BUCKET")),
		AccessKeyID:       strings.TrimSpace(os.Getenv("S3_ACCESS_KEY_ID")),
		SecretAccessKey:   strings.TrimSpace(os.Getenv("S3_SECRET_ACCESS_KEY")),
		PublicBaseURL:     strings.TrimSpace(os.Getenv("S3_PUBLIC_BASE_URL")),
		PresignTTLSeconds: s3PresignTTL,
		PreferPublicURL:   s3PreferPublicURL,
	}

	blobCfg := BlobConfig{
		Mode:           blobMode,
		ReportsMode:    reportsMode,
		ReportsModeSet: reportsModeSet,
		S3:             s3Cfg,
	}

	// REPORTS_MAX_RANGE_DAYS (default: 90)
	reportsMaxRangeDays := envInt("REPORTS_MAX_RANGE_DAYS", 90)

	// REPORTS_DEFAULT_TTL_HOURS (default: 168)
	reportsDefaultTTL := envInt("REPORTS_DEFAULT_TTL_HOURS", 168)

	// UPLOAD_MAX_MB (default: 10)
	uploadMaxMB := envInt("UPLOAD_MAX_MB", 10)

	// UPLOAD_ALLOWED_MIME (default: image/jpeg,image/png,image/heic)
	uploadAllowedMime := os.Getenv("UPLOAD_ALLOWED_MIME")
	if uploadAllowedMime == "" {
		uploadAllowedMime = "image/jpeg,image/png,image/heic"
	}

	// SOURCES_MAX_PER_CHECKIN (default: 4)
	sourcesMaxPerCheckin := envInt("SOURCES_MAX_PER_CHECKIN", 4)

	// NOTIFICATIONS_MAX_PER_DAY (default: 4)
	notificationsMaxPerDay := envInt("NOTIFICATIONS_MAX_PER_DAY", 4)

	// DEFAULT_SLEEP_MIN_MINUTES (default: 420 = 7 hours)
	defaultSleepMinMinutes := envInt("DEFAULT_SLEEP_MIN_MINUTES", 420)

	// DEFAULT_STEPS_MIN (default: 6000)
	defaultStepsMin := envInt("DEFAULT_STEPS_MIN", 6000)

	// DEFAULT_ACTIVE_ENERGY_MIN_KCAL (default: 200)
	defaultActiveEnergyMinKcal := envInt("DEFAULT_ACTIVE_ENERGY_MIN_KCAL", 200)

	// INTAKES_MAX_WATER_ML_PER_DAY (default: 8000)
	intakesMaxWaterMlPerDay := envInt("INTAKES_MAX_WATER_ML_PER_DAY", 8000)

	// INTAKES_WATER_DEFAULT_ADD_ML (default: 250)
	intakesWaterDefaultAddMl := envInt("INTAKES_WATER_DEFAULT_ADD_ML", 250)

	// INTAKES_MAX_SUPPLEMENTS (default: 100)
	intakesMaxSupplements := envInt("INTAKES_MAX_SUPPLEMENTS", 100)

	// AUTH_MODE (default: none)
	authMode := strings.ToLower(strings.TrimSpace(os.Getenv("AUTH_MODE")))
	if authMode == "" {
		// Backward-compat for legacy AUTH_ENABLED
		if authStr := strings.ToLower(strings.TrimSpace(os.Getenv("AUTH_ENABLED"))); authStr == "1" || authStr == "true" {
			authMode = "dev"
		} else {
			authMode = "none"
		}
	}
	if authMode != "none" && authMode != "dev" && authMode != "siwa" {
		log.Printf("WARNING: unknown AUTH_MODE=%q, fallback to none", authMode)
		authMode = "none"
	}
	authEnabled := authMode != "none"
	authRequired := authMode != "none" && (os.Getenv("AUTH_REQUIRED") == "1" || strings.EqualFold(os.Getenv("AUTH_REQUIRED"), "true"))

	// EMAIL_AUTH_ENABLED: if explicitly set, respect it; otherwise derive from AUTH_MODE.
	// When AUTH_MODE != "none", Email OTP is enabled by default.
	emailAuthEnabledRaw := strings.TrimSpace(os.Getenv("EMAIL_AUTH_ENABLED"))
	var emailAuthEnabled bool
	if emailAuthEnabledRaw != "" {
		emailAuthEnabled = parseBoolEnv("EMAIL_AUTH_ENABLED")
	} else {
		emailAuthEnabled = authEnabled // auto: true when AUTH_MODE is dev or siwa
	}

	// JWT_SECRET
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "change_me"
	}
	otpSecret := strings.TrimSpace(os.Getenv("OTP_SECRET"))
	if otpSecret == "" {
		otpSecret = jwtSecret
	}
	// Warn if using default in non-local environment
	if jwtSecret == "change_me" && env != "local" {
		log.Println("WARNING: JWT_SECRET is set to 'change_me' in non-local environment!")
	}

	// JWT_ISSUER (default: "health-hub")
	jwtIssuer := os.Getenv("JWT_ISSUER")
	if jwtIssuer == "" {
		jwtIssuer = "health-hub"
	}

	// JWT_TTL_MINUTES (default: 10080 = 7 days)
	jwtTTLMinutes := envInt("JWT_TTL_MINUTES", 10080)

	// OTP settings
	otpTTLSeconds := envInt("OTP_TTL_SECONDS", 600)
	if otpTTLSeconds <= 0 {
		otpTTLSeconds = 600
	}
	otpMaxAttempts := envInt("OTP_MAX_ATTEMPTS", 5)
	if otpMaxAttempts <= 0 {
		otpMaxAttempts = 5
	}
	otpResendMinSeconds := envInt("OTP_RESEND_MIN_SECONDS", 60)
	if otpResendMinSeconds <= 0 {
		otpResendMinSeconds = 60
	}
	otpMaxSendPerHour := envInt("OTP_MAX_SEND_PER_HOUR", 5)
	if otpMaxSendPerHour <= 0 {
		otpMaxSendPerHour = 5
	}
	emailSenderMode := strings.ToLower(strings.TrimSpace(os.Getenv("EMAIL_SENDER_MODE")))
	if emailSenderMode == "" {
		emailSenderMode = "local"
	}
	if emailSenderMode != "local" && emailSenderMode != "smtp" && emailSenderMode != "resend" {
		log.Printf("WARNING: unknown EMAIL_SENDER_MODE=%q, fallback to local", emailSenderMode)
		emailSenderMode = "local"
	}
	resendAPIKey := strings.TrimSpace(os.Getenv("RESEND_API_KEY"))
	resendFrom := strings.TrimSpace(os.Getenv("RESEND_FROM"))
	if resendFrom == "" {
		resendFrom = "HealthHub <onboarding@resend.dev>"
	}
	if emailSenderMode == "resend" && resendAPIKey == "" {
		log.Fatal("RESEND_API_KEY is required when EMAIL_SENDER_MODE=resend")
	}
	smtpHost := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	smtpPort := envInt("SMTP_PORT", 587)
	if smtpPort <= 0 {
		smtpPort = 587
	}
	smtpUsername := strings.TrimSpace(os.Getenv("SMTP_USERNAME"))
	smtpPassword := strings.TrimSpace(os.Getenv("SMTP_PASSWORD"))
	smtpFrom := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if smtpFrom == "" {
		smtpFrom = "HealthHub <no-reply@yourdomain.com>"
	}
	smtpUseTLS := parseBoolEnv("SMTP_USE_TLS")
	otpDebugReturnCode := parseBoolEnv("OTP_DEBUG_RETURN_CODE")

	// Apple Sign-In config
	appleBundleID := strings.TrimSpace(os.Getenv("APPLE_BUNDLE_ID"))
	if appleBundleID == "" {
		// Backward-compat for older env key.
		appleBundleID = strings.TrimSpace(os.Getenv("APPLE_AUDIENCE_BUNDLE_ID"))
	}

	appleIssuer := strings.TrimSpace(os.Getenv("APPLE_ISS"))
	if appleIssuer == "" {
		appleIssuer = "https://appleid.apple.com"
	}

	appleJWKSURL := strings.TrimSpace(os.Getenv("APPLE_JWKS_URL"))
	if appleJWKSURL == "" {
		appleJWKSURL = "https://appleid.apple.com/auth/keys"
	}

	appleSubPrefix := strings.TrimSpace(os.Getenv("APPLE_SUB_PREFIX"))
	if appleSubPrefix == "" {
		appleSubPrefix = "apple:"
	}

	if authMode == "siwa" && authRequired && appleBundleID == "" {
		log.Fatal("APPLE_BUNDLE_ID is required when AUTH_MODE=siwa and AUTH_REQUIRED=1")
	}

	// ---------- AI ----------
	aiMode := strings.ToLower(strings.TrimSpace(os.Getenv("AI_MODE")))
	if aiMode == "" {
		aiMode = "mock"
	}
	if aiMode != "mock" && aiMode != "openai" {
		log.Printf("WARNING: unknown AI_MODE=%q, fallback to mock", aiMode)
		aiMode = "mock"
	}

	aiMaxOutputTokens := envInt("AI_MAX_OUTPUT_TOKENS", 600)
	if aiMaxOutputTokens <= 0 {
		aiMaxOutputTokens = 600
	}

	aiTemperature := envFloat("AI_TEMPERATURE", 0.3)
	if aiTemperature < 0 {
		aiTemperature = 0
	}
	if aiTemperature > 2 {
		aiTemperature = 2
	}

	aiTimeoutSeconds := envInt("AI_TIMEOUT_SECONDS", 20)
	if aiTimeoutSeconds <= 0 {
		aiTimeoutSeconds = 20
	}

	openAIAPIKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	openAIModel := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if openAIModel == "" {
		openAIModel = "gpt-4.1-mini"
	}

	if aiMode == "openai" && openAIAPIKey == "" {
		log.Fatal("OPENAI_API_KEY is required when AI_MODE=openai")
	}

	return &Config{
		Env:               env,
		Port:              port,
		LogLevel:          logLevel,
		DatabaseURL:       runtimeDB,
		DatabaseURLRaw:    dbURL,
		DatabaseURLPooled: dbPooled,
		DatabaseURLDirect: dbDirect,

		CORSAllowedOrigins:   corsOrigins,
		CORSAllowCredentials: corsAllowCreds,

		RateLimitRPS:   rateLimitRPS,
		RateLimitBurst: rateLimitBurst,

		S3Endpoint:        s3Cfg.Endpoint,
		S3Region:          s3Cfg.Region,
		S3Bucket:          s3Cfg.Bucket,
		S3AccessKeyID:     s3Cfg.AccessKeyID,
		S3SecretAccessKey: s3Cfg.SecretAccessKey,
		S3PublicBaseURL:   s3Cfg.PublicBaseURL,
		S3PresignTTL:      s3Cfg.PresignTTLSeconds,
		Blob:              blobCfg,

		ReportsMaxRangeDays:    reportsMaxRangeDays,
		ReportsDefaultTTLHours: reportsDefaultTTL,

		UploadMaxMB:          uploadMaxMB,
		UploadAllowedMime:    uploadAllowedMime,
		SourcesMaxPerCheckin: sourcesMaxPerCheckin,

		NotificationsMaxPerDay:     notificationsMaxPerDay,
		DefaultSleepMinMinutes:     defaultSleepMinMinutes,
		DefaultStepsMin:            defaultStepsMin,
		DefaultActiveEnergyMinKcal: defaultActiveEnergyMinKcal,

		IntakesMaxWaterMlPerDay:  intakesMaxWaterMlPerDay,
		IntakesWaterDefaultAddMl: intakesWaterDefaultAddMl,
		IntakesMaxSupplements:    intakesMaxSupplements,

		AuthMode:            authMode,
		AuthEnabled:         authEnabled,
		AuthRequired:        authRequired,
		EmailAuthEnabled:    emailAuthEnabled,
		JWTSecret:           jwtSecret,
		OTPSecret:           otpSecret,
		JWTIssuer:           jwtIssuer,
		JWTTTLMinutes:       jwtTTLMinutes,
		OTPTTLSeconds:       otpTTLSeconds,
		OTPMaxAttempts:      otpMaxAttempts,
		OTPResendMinSeconds: otpResendMinSeconds,
		OTPMaxSendPerHour:   otpMaxSendPerHour,
		AppleBundleID:       appleBundleID,
		AppleIssuer:         appleIssuer,
		AppleJWKSURL:        appleJWKSURL,
		AppleSubPrefix:      appleSubPrefix,
		EmailSenderMode:     emailSenderMode,
		SMTPHost:            smtpHost,
		SMTPPort:            smtpPort,
		SMTPUsername:        smtpUsername,
		SMTPPassword:        smtpPassword,
		SMTPFrom:            smtpFrom,
		SMTPUseTLS:          smtpUseTLS,
		ResendAPIKey:        resendAPIKey,
		ResendFrom:          resendFrom,
		OTPDebugReturnCode:  otpDebugReturnCode,
		AIMode:              aiMode,
		AIMaxOutputTokens:   aiMaxOutputTokens,
		AITemperature:       aiTemperature,
		AITimeoutSeconds:    aiTimeoutSeconds,
		OpenAIAPIKey:        openAIAPIKey,
		OpenAIModel:         openAIModel,

		RunMigrationsOnStartup: runMigrationsOnStartup,
	}
}

// parseCORSOrigins parses CORS_ALLOWED_ORIGINS env var.
// In local mode, defaults to localhost origins if empty.
func parseCORSOrigins(raw, env string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if env == "local" {
			return []string{"http://localhost:3000", "http://localhost:8081"}
		}
		return nil // prod: deny by default
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			origins = append(origins, p)
		}
	}
	return origins
}

func parseBlobMode(key string, defaultVal string) string {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if mode == "" {
		return defaultVal
	}
	switch mode {
	case BlobModeLocal, BlobModeS3, BlobModeAuto:
		return mode
	default:
		log.Printf("WARNING: unknown %s=%q, fallback to %s", key, mode, defaultVal)
		return defaultVal
	}
}

// envInt reads an int env var with a default value.
func envInt(key string, defaultVal int) int {
	s := os.Getenv(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

func envFloat(key string, defaultVal float64) float64 {
	s := os.Getenv(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defaultVal
	}
	return v
}

func parseBoolEnv(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
