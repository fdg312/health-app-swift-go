package blob

import (
	"fmt"
	"strings"

	appcfg "github.com/fdg312/health-hub/internal/config"
)

type Logger interface {
	Printf(format string, v ...any)
}

// NewBlobStore builds a blob store using mode local|s3|auto.
func NewBlobStore(cfg appcfg.BlobConfig, logger Logger) (Store, string, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	if mode == "" {
		mode = appcfg.BlobModeLocal
	}

	switch mode {
	case appcfg.BlobModeLocal:
		logf(logger, "INFO blob: mode=local (forced)")
		return nil, appcfg.BlobModeLocal, nil

	case appcfg.BlobModeAuto:
		if !cfg.S3.IsConfigured() {
			level, code, msg := cfg.S3.Diagnostics()
			summary := cfg.S3.DiagnosticsSummary()
			logf(logger, "%s blob.s3: code=%s %s", level, code, msg)
			logf(logger, "INFO blob.s3: %s", summary)
			logf(logger, "INFO blob: mode=local (auto, S3 not configured)")
			return nil, appcfg.BlobModeLocal, nil
		}

		summary := cfg.S3.DiagnosticsSummary()
		logf(logger, "INFO blob.s3: code=s3_ready %s", summary)
		store, err := NewS3Store(cfg.S3.Endpoint, cfg.S3.Region, cfg.S3.Bucket, cfg.S3.AccessKeyID, cfg.S3.SecretAccessKey)
		if err != nil {
			logf(logger, "WARN blob.s3: init_failed=%q, fallback=local", err.Error())
			return nil, appcfg.BlobModeLocal, nil
		}

		logf(logger, "INFO blob: mode=s3 (auto, configured)")
		return store, appcfg.BlobModeS3, nil

	case appcfg.BlobModeS3:
		if !cfg.S3.IsConfigured() {
			missing := cfg.S3.MissingRequired()
			summary := cfg.S3.DiagnosticsSummary()
			logf(logger, "FATAL blob.s3: code=s3_config_incomplete missing=%v", missing)
			logf(logger, "FATAL blob.s3: %s", summary)
			err := fmt.Errorf("BLOB_MODE=s3 requested but missing required config: %s", strings.Join(missing, ", "))
			return nil, "", err
		}

		summary := cfg.S3.DiagnosticsSummary()
		logf(logger, "INFO blob.s3: code=s3_ready %s", summary)
		store, err := NewS3Store(cfg.S3.Endpoint, cfg.S3.Region, cfg.S3.Bucket, cfg.S3.AccessKeyID, cfg.S3.SecretAccessKey)
		if err != nil {
			logf(logger, "FATAL blob.s3: init_failed=%v", err)
			return nil, "", fmt.Errorf("BLOB_MODE=s3 init failed: %w", err)
		}

		logf(logger, "INFO blob: mode=s3 (forced)")
		return store, appcfg.BlobModeS3, nil

	default:
		return nil, "", fmt.Errorf("unsupported blob mode: %s", mode)
	}
}

func logf(logger Logger, format string, v ...any) {
	if logger == nil {
		return
	}
	logger.Printf(format, v...)
}
