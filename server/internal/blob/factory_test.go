package blob

import (
	"bytes"
	"log"
	"strings"
	"testing"

	appcfg "github.com/fdg312/health-hub/internal/config"
)

func TestNewBlobStoreLocalForced(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	store, mode, err := NewBlobStore(appcfg.BlobConfig{
		Mode: appcfg.BlobModeLocal,
		S3:   appcfg.S3Config{},
	}, logger)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mode != appcfg.BlobModeLocal {
		t.Fatalf("expected mode=local, got %s", mode)
	}
	if store != nil {
		t.Fatal("expected nil store in local mode")
	}
	if !strings.Contains(buf.String(), "mode=local (forced)") {
		t.Fatalf("expected local mode log, got: %s", buf.String())
	}
}

func TestNewBlobStoreAutoEmptyS3FallsBackToLocal(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	store, mode, err := NewBlobStore(appcfg.BlobConfig{
		Mode: appcfg.BlobModeAuto,
		S3:   appcfg.S3Config{},
	}, logger)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mode != appcfg.BlobModeLocal {
		t.Fatalf("expected mode=local fallback, got %s", mode)
	}
	if store != nil {
		t.Fatal("expected nil store on auto fallback")
	}

	logOut := buf.String()
	if !strings.Contains(logOut, "code=s3_not_configured") {
		t.Fatalf("expected s3_not_configured diagnostics, got: %s", logOut)
	}
	if !strings.Contains(logOut, "mode=local (auto, S3 not configured)") {
		t.Fatalf("expected auto fallback to local log, got: %s", logOut)
	}
}

func TestNewBlobStoreS3MissingRequiredReturnsError(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	store, mode, err := NewBlobStore(appcfg.BlobConfig{
		Mode: appcfg.BlobModeS3,
		S3: appcfg.S3Config{
			Endpoint: "https://storage.yandexcloud.net",
		},
	}, logger)
	if err == nil {
		t.Fatal("expected error when mode=s3 and required env are missing")
	}
	if store != nil || mode != "" {
		t.Fatalf("expected nil store and empty mode on error, got store=%v mode=%q", store, mode)
	}
	if !strings.Contains(err.Error(), "missing required config") {
		t.Fatalf("expected missing required config error, got: %v", err)
	}
}
