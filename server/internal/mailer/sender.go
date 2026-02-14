package mailer

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/fdg312/health-hub/internal/config"
)

// Sender delivers plain-text email messages.
type Sender interface {
	Send(to, subject, textBody string) error
}

// NewSenderFromConfig builds email sender based on config.
func NewSenderFromConfig(cfg *config.Config, logger *log.Logger) (Sender, error) {
	if logger == nil {
		logger = log.Default()
	}

	mode := strings.ToLower(strings.TrimSpace(cfg.EmailSenderMode))
	if mode == "" {
		mode = "local"
	}

	switch mode {
	case "local":
		return NewLocalSender(logger), nil
	case "smtp":
		if strings.TrimSpace(cfg.SMTPHost) == "" {
			return nil, errors.New("SMTP_HOST is required for EMAIL_SENDER_MODE=smtp")
		}
		if cfg.SMTPPort <= 0 {
			return nil, errors.New("SMTP_PORT must be greater than 0 for EMAIL_SENDER_MODE=smtp")
		}
		if strings.TrimSpace(cfg.SMTPFrom) == "" {
			return nil, errors.New("SMTP_FROM is required for EMAIL_SENDER_MODE=smtp")
		}
		if strings.TrimSpace(cfg.SMTPUsername) != "" && strings.TrimSpace(cfg.SMTPPassword) == "" {
			return nil, errors.New("SMTP_PASSWORD is required when SMTP_USERNAME is set")
		}

		return NewSMTPSender(SMTPConfig{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			Username: cfg.SMTPUsername,
			Password: cfg.SMTPPassword,
			From:     cfg.SMTPFrom,
			UseTLS:   cfg.SMTPUseTLS,
		}), nil
	case "resend":
		if strings.TrimSpace(cfg.ResendAPIKey) == "" {
			return nil, errors.New("RESEND_API_KEY is required for EMAIL_SENDER_MODE=resend")
		}
		return NewResendSender(ResendConfig{
			APIKey: cfg.ResendAPIKey,
			From:   cfg.ResendFrom,
		}), nil
	default:
		return nil, fmt.Errorf("unsupported EMAIL_SENDER_MODE=%q", mode)
	}
}
