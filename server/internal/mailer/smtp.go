package mailer

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
)

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool
}

type SMTPSender struct {
	cfg SMTPConfig
}

func NewSMTPSender(cfg SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) Send(to, subject, textBody string) error {
	addr := net.JoinHostPort(s.cfg.Host, fmt.Sprintf("%d", s.cfg.Port))

	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server %s: %w", addr, err)
	}
	if client != nil {
		defer client.Close()
	}

	if s.cfg.UseTLS {
		ok, _ := client.Extension("STARTTLS")
		if !ok {
			return fmt.Errorf("smtp server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{
			ServerName: s.cfg.Host,
			MinVersion: tls.VersionTLS12,
		}); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	if strings.TrimSpace(s.cfg.Username) != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp authentication failed: %w", err)
		}
	}

	fromAddress, err := envelopeAddress(s.cfg.From)
	if err != nil {
		return err
	}

	if err := client.Mail(fromAddress); err != nil {
		return fmt.Errorf("smtp MAIL command failed: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp RCPT command failed for %s: %w", to, err)
	}

	dataWriter, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA command failed: %w", err)
	}

	message := buildMessage(s.cfg.From, to, subject, textBody)
	if _, err := dataWriter.Write([]byte(message)); err != nil {
		_ = dataWriter.Close()
		return err
	}
	if err := dataWriter.Close(); err != nil {
		return err
	}

	return client.Quit()
}

func envelopeAddress(from string) (string, error) {
	parsed, err := mail.ParseAddress(strings.TrimSpace(from))
	if err != nil {
		return "", fmt.Errorf("invalid SMTP_FROM: %w", err)
	}
	return parsed.Address, nil
}

func buildMessage(from, to, subject, body string) string {
	safeSubject := strings.ReplaceAll(strings.ReplaceAll(subject, "\r", " "), "\n", " ")
	safeTo := strings.ReplaceAll(strings.ReplaceAll(to, "\r", ""), "\n", "")

	headers := []string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", safeTo),
		fmt.Sprintf("Subject: %s", safeSubject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
	}

	return strings.Join(headers, "\r\n") + body
}
