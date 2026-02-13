package dbmigrate

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Run(command string, dbURL string, migrationsDir string) error {
	if dbURL == "" {
		return fmt.Errorf("database URL is empty")
	}
	if migrationsDir == "" {
		migrationsDir = DefaultMigrationsDir
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Run(command, db, migrationsDir); err != nil {
		return fmt.Errorf("goose %s failed: %w", command, err)
	}

	return nil
}
