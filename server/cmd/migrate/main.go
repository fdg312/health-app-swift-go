package main

import (
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"

	"github.com/fdg312/health-hub/internal/config"
	"github.com/fdg312/health-hub/internal/dbmigrate"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: go run ./cmd/migrate [up|status|down]")
	}

	command := os.Args[1]
	switch command {
	case "up", "status", "down":
	default:
		log.Fatalf("unsupported command %q (allowed: up, status, down)", command)
	}

	cfg := config.Load()
	dbURL, source, warning, err := dbmigrate.SelectDatabaseURL(cfg, false)
	if err != nil {
		log.Fatal(err)
	}

	if warning != "" {
		log.Printf("WARN migrate: %s", warning)
	}
	log.Printf("migrate: command=%s using=%s", command, source)

	if err := dbmigrate.Run(command, dbURL, dbmigrate.DefaultMigrationsDir); err != nil {
		log.Fatal(err)
	}

	log.Printf("migrate: %s completed successfully", command)
}
