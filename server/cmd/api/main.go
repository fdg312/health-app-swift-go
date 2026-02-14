package main

import (
	"fmt"
	"log"
	"strings"

	_ "github.com/joho/godotenv/autoload"

	"github.com/fdg312/health-hub/internal/config"
	"github.com/fdg312/health-hub/internal/dbmigrate"
	"github.com/fdg312/health-hub/internal/httpserver"
)

func main() {
	cfg := config.Load()

	printStartupBanner(cfg)

	if cfg.RunMigrationsOnStartup {
		dbURL, source, _, err := dbmigrate.SelectDatabaseURL(cfg, true)
		if err != nil {
			log.Fatalf("FATAL startup migrations: %v", err)
		}

		log.Printf("startup migrations: command=up using=%s", source)
		if err := dbmigrate.Run("up", dbURL, dbmigrate.DefaultMigrationsDir); err != nil {
			log.Fatalf("FATAL startup migrations failed: %v", err)
		}
		log.Printf("startup migrations: completed")
	}

	validateProductionConfig(cfg)

	server := httpserver.New(cfg)

	log.Fatal(server.Start())
}

// printStartupBanner logs a one-time summary of the resolved configuration.
// No secrets are ever printed — only masked indicators ("set" / "not set").
func printStartupBanner(cfg *config.Config) {
	log.Println("========== Health Hub API ==========")
	log.Printf("  env              = %s", cfg.Env)
	log.Printf("  port             = %d", cfg.Port)

	// ---- Database ----
	log.Println("---- database ----")
	log.Printf("  runtime_url      = %s", describeDBURL(cfg.DatabaseURL, cfg.DatabaseURLPooled))
	log.Printf("  pooled           = %s", setOrNot(cfg.DatabaseURLPooled))
	log.Printf("  direct           = %s", setOrNot(cfg.DatabaseURLDirect))
	log.Printf("  migrations_on_startup = %t", cfg.RunMigrationsOnStartup)
	if cfg.RunMigrationsOnStartup {
		if cfg.DatabaseURLDirect != "" {
			log.Printf("  migrations_via   = DATABASE_URL_DIRECT")
		} else {
			log.Printf("  migrations_via   = (will fail — DATABASE_URL_DIRECT not set)")
		}
	}

	// ---- Auth ----
	log.Println("---- auth ----")
	log.Printf("  auth_mode        = %s", cfg.AuthMode)
	log.Printf("  auth_required    = %t", cfg.AuthRequired)
	log.Printf("  email_auth       = %t", cfg.EmailAuthEnabled)
	log.Printf("  jwt_secret       = %s", secretStatus(cfg.JWTSecret, "change_me"))
	log.Printf("  otp_secret       = %s", setOrNot(cfg.OTPSecret))
	if cfg.AuthMode == "siwa" {
		log.Printf("  apple_bundle_id  = %s", nonEmptyOrDash(cfg.AppleBundleID))
	}

	// ---- Blob / S3 ----
	log.Println("---- blob ----")
	log.Printf("  blob_mode        = %s", cfg.Blob.Mode)
	log.Printf("  reports_mode     = %s (effective=%s)", displayReportsMode(cfg), cfg.Blob.EffectiveReportsMode())
	if cfg.Blob.Mode != config.BlobModeLocal || cfg.Blob.EffectiveReportsMode() != config.BlobModeLocal {
		log.Printf("  s3: %s", cfg.Blob.S3.DiagnosticsSummary())
	}

	// ---- Mailer ----
	log.Println("---- mailer ----")
	log.Printf("  email_sender     = %s", cfg.EmailSenderMode)
	switch cfg.EmailSenderMode {
	case "smtp":
		log.Printf("  smtp_host        = %s", nonEmptyOrDash(cfg.SMTPHost))
		log.Printf("  smtp_port        = %d", cfg.SMTPPort)
		log.Printf("  smtp_from        = %s", nonEmptyOrDash(cfg.SMTPFrom))
		log.Printf("  smtp_username    = %s", setOrNot(cfg.SMTPUsername))
		log.Printf("  smtp_password    = %s", setOrNot(cfg.SMTPPassword))
		log.Printf("  smtp_use_tls     = %t", cfg.SMTPUseTLS)
	case "resend":
		log.Printf("  resend_api_key   = %s", setOrNot(cfg.ResendAPIKey))
		log.Printf("  resend_from      = %s", nonEmptyOrDash(cfg.ResendFrom))
	default:
		log.Printf("  (OTP codes will be printed to the server console)")
	}

	// ---- AI ----
	log.Println("---- ai ----")
	log.Printf("  ai_mode          = %s", cfg.AIMode)
	if cfg.AIMode == "openai" {
		log.Printf("  openai_model     = %s", cfg.OpenAIModel)
		log.Printf("  openai_api_key   = %s", setOrNot(cfg.OpenAIAPIKey))
	}

	log.Println("====================================")
}

// validateProductionConfig performs fatal checks that only matter in non-local envs.
func validateProductionConfig(cfg *config.Config) {
	isProd := cfg.Env == "production" || cfg.Env == "staging"

	// S3 hard-mode validation
	needsS3 := cfg.Blob.Mode == config.BlobModeS3 || cfg.Blob.EffectiveReportsMode() == config.BlobModeS3
	if needsS3 {
		if missing := cfg.Blob.S3.MissingRequired(); len(missing) > 0 {
			log.Fatalf("FATAL blob: BLOB_MODE or REPORTS_MODE is 's3' but S3 config is incomplete — missing: %s", strings.Join(missing, ", "))
		}
	}

	// SMTP validation when enabled
	if cfg.EmailSenderMode == "smtp" {
		var missing []string
		if strings.TrimSpace(cfg.SMTPHost) == "" {
			missing = append(missing, "SMTP_HOST")
		}
		if cfg.SMTPPort <= 0 {
			missing = append(missing, "SMTP_PORT")
		}
		if strings.TrimSpace(cfg.SMTPFrom) == "" {
			missing = append(missing, "SMTP_FROM")
		}
		if len(missing) > 0 {
			log.Fatalf("FATAL mailer: EMAIL_SENDER_MODE=smtp but config is incomplete — missing: %s", strings.Join(missing, ", "))
		}
	}

	// Resend validation when enabled
	if cfg.EmailSenderMode == "resend" {
		if strings.TrimSpace(cfg.ResendAPIKey) == "" {
			log.Fatal("FATAL mailer: EMAIL_SENDER_MODE=resend but RESEND_API_KEY is not set")
		}
	}

	// JWT_SECRET must not be default in production
	if isProd && cfg.AuthRequired && cfg.JWTSecret == "change_me" {
		log.Fatalf("FATAL auth: JWT_SECRET must not be 'change_me' in %s with AUTH_REQUIRED=1", cfg.Env)
	}

	// DATABASE_URL must be set in production
	if isProd && cfg.DatabaseURL == "" {
		log.Fatalf("FATAL db: no DATABASE_URL configured in %s", cfg.Env)
	}
}

// ---- helpers (no secrets) ----

func setOrNot(v string) string {
	if strings.TrimSpace(v) == "" {
		return "not set"
	}
	return "set"
}

func nonEmptyOrDash(v string) string {
	if strings.TrimSpace(v) == "" {
		return "-"
	}
	return v
}

func secretStatus(v, insecureDefault string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "not set"
	}
	if v == insecureDefault {
		return fmt.Sprintf("set (DEFAULT — insecure '%s')", insecureDefault)
	}
	return "set (custom)"
}

func describeDBURL(runtime, pooled string) string {
	if runtime == "" {
		return "not set (will use in-memory storage)"
	}
	if pooled != "" && runtime == pooled {
		return "set (via DATABASE_URL_POOLED)"
	}
	return "set"
}

func displayReportsMode(cfg *config.Config) string {
	if cfg.Blob.ReportsModeSet {
		return cfg.Blob.ReportsMode
	}
	return fmt.Sprintf("(inherits BLOB_MODE=%s)", cfg.Blob.Mode)
}
