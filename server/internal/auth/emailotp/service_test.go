package emailotp

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fdg312/health-hub/internal/config"
	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/storage/memory"
)

type mockSender struct {
	to      string
	subject string
	body    string
	calls   int
}

func (m *mockSender) Send(to, subject, textBody string) error {
	m.to = to
	m.subject = subject
	m.body = textBody
	m.calls++
	return nil
}

type testHarness struct {
	service *Service
	store   storage.EmailOTPStorage
	sender  *mockSender
	now     *time.Time
}

func newHarness(t *testing.T, emailEnabled bool) *testHarness {
	t.Helper()

	mem := memory.New()
	sender := &mockSender{}
	cfg := &config.Config{
		Env:                 "local",
		EmailAuthEnabled:    emailEnabled,
		JWTSecret:           "test-jwt-secret",
		OTPSecret:           "test-otp-secret",
		JWTIssuer:           "health-hub-test",
		OTPTTLSeconds:       600,
		OTPMaxAttempts:      2,
		OTPResendMinSeconds: 60,
		OTPMaxSendPerHour:   5,
		OTPDebugReturnCode:  true,
	}

	now := time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC)
	svc := NewService(cfg, mem, sender)
	svc.now = func() time.Time { return now }
	svc.generateCode = func() (string, error) { return "123456", nil }

	return &testHarness{
		service: svc,
		store:   mem,
		sender:  sender,
		now:     &now,
	}
}

func TestRequestCreatesOTPAndSendsEmail(t *testing.T) {
	h := newHarness(t, true)

	resp, err := h.service.Request(context.Background(), " User@Example.com ")
	if err != nil {
		t.Fatalf("request otp failed: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("expected status=ok, got %q", resp.Status)
	}
	if resp.DebugCode == nil || *resp.DebugCode != "123456" {
		t.Fatalf("expected debug_code=123456, got %v", resp.DebugCode)
	}
	if h.sender.calls != 1 {
		t.Fatalf("expected sender calls=1, got %d", h.sender.calls)
	}
	if h.sender.to != "user@example.com" {
		t.Fatalf("expected normalized email, got %q", h.sender.to)
	}
	if !strings.Contains(h.sender.body, "123456") {
		t.Fatalf("expected code in body, got %q", h.sender.body)
	}

	otp, err := h.store.GetLatestActive(context.Background(), "user@example.com", *h.now)
	if err != nil {
		t.Fatalf("get active otp failed: %v", err)
	}
	if otp == nil {
		t.Fatal("expected active otp")
	}
}

func TestVerifyCorrectCodeReturnsAccessToken(t *testing.T) {
	h := newHarness(t, true)

	if _, err := h.service.Request(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("request otp failed: %v", err)
	}

	resp, err := h.service.Verify(context.Background(), "user@example.com", "123456")
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if resp.AccessToken == "" {
		t.Fatal("expected access token")
	}
	if resp.UserID != "email:user@example.com" {
		t.Fatalf("expected user_id=email:user@example.com, got %q", resp.UserID)
	}
}

func TestVerifyWrongCodeIncrementsAttempts(t *testing.T) {
	h := newHarness(t, true)

	if _, err := h.service.Request(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("request otp failed: %v", err)
	}

	_, err := h.service.Verify(context.Background(), "user@example.com", "000000")
	if err == nil {
		t.Fatal("expected error for wrong code")
	}

	serviceErr, ok := AsServiceError(err)
	if !ok || serviceErr.Code != "otp_invalid_code" {
		t.Fatalf("expected otp_invalid_code, got %v", err)
	}

	otp, err := h.store.GetLatestActive(context.Background(), "user@example.com", *h.now)
	if err != nil {
		t.Fatalf("get active otp failed: %v", err)
	}
	if otp == nil {
		t.Fatal("expected active otp")
	}
	if otp.Attempts != 1 {
		t.Fatalf("expected attempts=1, got %d", otp.Attempts)
	}
}

func TestVerifyLocksAfterMaxAttempts(t *testing.T) {
	h := newHarness(t, true)

	if _, err := h.service.Request(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("request otp failed: %v", err)
	}

	_, _ = h.service.Verify(context.Background(), "user@example.com", "000000")
	_, _ = h.service.Verify(context.Background(), "user@example.com", "000000")
	_, err := h.service.Verify(context.Background(), "user@example.com", "123456")
	if err == nil {
		t.Fatal("expected locked error")
	}

	serviceErr, ok := AsServiceError(err)
	if !ok || serviceErr.Code != "otp_locked" {
		t.Fatalf("expected otp_locked, got %v", err)
	}
}

func TestRequestResendTooSoon(t *testing.T) {
	h := newHarness(t, true)

	if _, err := h.service.Request(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("request otp failed: %v", err)
	}

	_, err := h.service.Request(context.Background(), "user@example.com")
	if err == nil {
		t.Fatal("expected resend rate limit")
	}

	serviceErr, ok := AsServiceError(err)
	if !ok || serviceErr.Code != "otp_resend_too_soon" {
		t.Fatalf("expected otp_resend_too_soon, got %v", err)
	}
}

func TestRequestRateLimitedByHourlyQuota(t *testing.T) {
	h := newHarness(t, true)
	h.service.cfg.OTPResendMinSeconds = 1
	h.service.cfg.OTPMaxSendPerHour = 2

	if _, err := h.service.Request(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("first request failed: %v", err)
	}

	*h.now = h.now.Add(2 * time.Second)
	if _, err := h.service.Request(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("second request failed: %v", err)
	}

	*h.now = h.now.Add(2 * time.Second)
	_, err := h.service.Request(context.Background(), "user@example.com")
	if err == nil {
		t.Fatal("expected otp_rate_limited")
	}

	serviceErr, ok := AsServiceError(err)
	if !ok || serviceErr.Code != "otp_rate_limited" {
		t.Fatalf("expected otp_rate_limited, got %v", err)
	}
}

func TestRequestWhenDisabled(t *testing.T) {
	h := newHarness(t, false)

	_, err := h.service.Request(context.Background(), "user@example.com")
	if err == nil {
		t.Fatal("expected disabled error")
	}

	serviceErr, ok := AsServiceError(err)
	if !ok || serviceErr.Code != "email_auth_disabled" {
		t.Fatalf("expected email_auth_disabled, got %v", err)
	}
}

func TestHashCodeStable(t *testing.T) {
	hash1 := HashCode("user@example.com", "123456", "secret")
	hash2 := HashCode(" User@Example.com ", "123456", "secret")
	if hash1 != hash2 {
		t.Fatalf("expected stable hash for normalized email")
	}

	hash3 := HashCode("user@example.com", "654321", "secret")
	if hash1 == hash3 {
		t.Fatal("expected different hashes for different code")
	}
}
