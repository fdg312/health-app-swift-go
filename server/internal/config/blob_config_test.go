package config

import "testing"

func TestS3ConfigIsConfigured(t *testing.T) {
	t.Run("empty config is not configured", func(t *testing.T) {
		cfg := S3Config{}
		if cfg.IsConfigured() {
			t.Fatal("expected IsConfigured=false for empty config")
		}
	})

	t.Run("required fields set is configured", func(t *testing.T) {
		cfg := S3Config{
			Endpoint:        "https://storage.yandexcloud.net",
			Region:          "ru-central1",
			Bucket:          "bucket",
			AccessKeyID:     "key",
			SecretAccessKey: "secret",
			PublicBaseURL:   "https://storage.yandexcloud.net/bucket",
		}
		if !cfg.IsConfigured() {
			t.Fatal("expected IsConfigured=true when all required fields are set")
		}
	})
}

func TestS3ConfigMissingRequired(t *testing.T) {
	cfg := S3Config{
		Endpoint: "https://storage.yandexcloud.net",
		Bucket:   "bucket",
	}
	missing := cfg.MissingRequired()

	want := []string{"S3_REGION", "S3_ACCESS_KEY_ID", "S3_SECRET_ACCESS_KEY", "S3_PUBLIC_BASE_URL"}
	if len(missing) != len(want) {
		t.Fatalf("expected %d missing fields, got %d (%v)", len(want), len(missing), missing)
	}
	for i := range want {
		if missing[i] != want[i] {
			t.Fatalf("expected missing[%d]=%s, got %s", i, want[i], missing[i])
		}
	}
}

func TestS3ConfigDiagnostics(t *testing.T) {
	t.Run("not configured", func(t *testing.T) {
		level, code, _ := (S3Config{}).Diagnostics()
		if level != "INFO" || code != "s3_not_configured" {
			t.Fatalf("expected INFO/s3_not_configured, got %s/%s", level, code)
		}
	})

	t.Run("partial config", func(t *testing.T) {
		level, code, _ := (S3Config{Endpoint: "https://storage.yandexcloud.net"}).Diagnostics()
		if level != "WARN" || code != "s3_partial_config" {
			t.Fatalf("expected WARN/s3_partial_config, got %s/%s", level, code)
		}
	})

	t.Run("region missing", func(t *testing.T) {
		level, code, _ := (S3Config{
			Endpoint:        "https://storage.yandexcloud.net",
			Bucket:          "bucket",
			AccessKeyID:     "key",
			SecretAccessKey: "secret",
		}).Diagnostics()
		if level != "WARN" || code != "s3_partial_config" {
			t.Fatalf("expected WARN/s3_partial_config (missing region+publicBaseURL), got %s/%s", level, code)
		}
	})

	t.Run("ready", func(t *testing.T) {
		level, code, _ := (S3Config{
			Endpoint:        "https://storage.yandexcloud.net",
			Region:          "ru-central1",
			Bucket:          "bucket",
			AccessKeyID:     "key",
			SecretAccessKey: "secret",
			PublicBaseURL:   "https://storage.yandexcloud.net/bucket",
		}).Diagnostics()
		if level != "INFO" || code != "s3_ready" {
			t.Fatalf("expected INFO/s3_ready, got %s/%s", level, code)
		}
	})
}
