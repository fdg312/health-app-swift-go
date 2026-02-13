package dbmigrate

import (
	"fmt"

	"github.com/fdg312/health-hub/internal/config"
)

const DefaultMigrationsDir = "migrations"

// SelectDatabaseURL selects DB URL for migrations.
// Priority for migration command: DIRECT > DATABASE_URL > POOLED (with warning).
// If requireDirect is true, only DATABASE_URL_DIRECT is accepted.
func SelectDatabaseURL(cfg *config.Config, requireDirect bool) (dbURL string, source string, warning string, err error) {
	if requireDirect {
		if cfg.DatabaseURLDirect == "" {
			return "", "", "", fmt.Errorf("DATABASE_URL_DIRECT is required for DDL/migrations")
		}
		return cfg.DatabaseURLDirect, "DATABASE_URL_DIRECT", "", nil
	}

	if cfg.DatabaseURLDirect != "" {
		return cfg.DatabaseURLDirect, "DATABASE_URL_DIRECT", "", nil
	}
	if cfg.DatabaseURLRaw != "" {
		return cfg.DatabaseURLRaw, "DATABASE_URL", "", nil
	}
	if cfg.DatabaseURLPooled != "" {
		return cfg.DatabaseURLPooled, "DATABASE_URL_POOLED", "using pooled connection for DDL is not recommended; set DATABASE_URL_DIRECT", nil
	}

	return "", "", "", fmt.Errorf("no database URL configured (set DATABASE_URL_DIRECT or DATABASE_URL)")
}
