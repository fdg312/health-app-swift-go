package dbmigrate

import (
	"testing"

	"github.com/fdg312/health-hub/internal/config"
)

func TestSelectDatabaseURL_Priority(t *testing.T) {
	cfg := &config.Config{
		DatabaseURLDirect: "postgres://direct",
		DatabaseURLRaw:    "postgres://url",
		DatabaseURLPooled: "postgres://pooled",
	}

	dbURL, source, warning, err := SelectDatabaseURL(cfg, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dbURL != "postgres://direct" || source != "DATABASE_URL_DIRECT" {
		t.Fatalf("expected direct URL, got dbURL=%q source=%q", dbURL, source)
	}
	if warning != "" {
		t.Fatalf("unexpected warning: %q", warning)
	}
}

func TestSelectDatabaseURL_FallbackToDatabaseURL(t *testing.T) {
	cfg := &config.Config{
		DatabaseURLRaw:    "postgres://url",
		DatabaseURLPooled: "postgres://pooled",
	}

	dbURL, source, warning, err := SelectDatabaseURL(cfg, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dbURL != "postgres://url" || source != "DATABASE_URL" {
		t.Fatalf("expected DATABASE_URL, got dbURL=%q source=%q", dbURL, source)
	}
	if warning != "" {
		t.Fatalf("unexpected warning: %q", warning)
	}
}

func TestSelectDatabaseURL_PooledWarning(t *testing.T) {
	cfg := &config.Config{
		DatabaseURLPooled: "postgres://pooled",
	}

	dbURL, source, warning, err := SelectDatabaseURL(cfg, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dbURL != "postgres://pooled" || source != "DATABASE_URL_POOLED" {
		t.Fatalf("expected pooled URL, got dbURL=%q source=%q", dbURL, source)
	}
	if warning == "" {
		t.Fatal("expected warning for pooled DDL usage")
	}
}

func TestSelectDatabaseURL_RequireDirect(t *testing.T) {
	cfg := &config.Config{
		DatabaseURLRaw:    "postgres://url",
		DatabaseURLPooled: "postgres://pooled",
	}

	_, _, _, err := SelectDatabaseURL(cfg, true)
	if err == nil {
		t.Fatal("expected error when direct is required but missing")
	}
}
