package emailotp

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fdg312/health-hub/internal/config"
	"github.com/fdg312/health-hub/internal/mailer"
	"github.com/fdg312/health-hub/internal/storage"
	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultTokenTTL = 30 * 24 * time.Hour
)

type Service struct {
	cfg     *config.Config
	storage storage.EmailOTPStorage
	sender  mailer.Sender

	now          func() time.Time
	generateCode func() (string, error)
}

func NewService(cfg *config.Config, otpStorage storage.EmailOTPStorage, sender mailer.Sender) *Service {
	return &Service{
		cfg:          cfg,
		storage:      otpStorage,
		sender:       sender,
		now:          time.Now,
		generateCode: GenerateCode,
	}
}

func (s *Service) Request(ctx context.Context, emailRaw string) (*RequestResponse, error) {
	if !s.cfg.EmailAuthEnabled {
		return nil, &ServiceError{
			Status:  http.StatusNotFound,
			Code:    "email_auth_disabled",
			Message: "Email auth is disabled",
		}
	}
	if s.storage == nil || s.sender == nil {
		return nil, errors.New("email otp service is not initialized")
	}

	email := normalizeEmail(emailRaw)
	if !isValidEmail(email) {
		return nil, &ServiceError{
			Status:  http.StatusBadRequest,
			Code:    "invalid_email",
			Message: "Invalid email format",
		}
	}

	now := s.now()
	latest, err := s.storage.GetLatestActive(ctx, email, now)
	if err != nil {
		return nil, err
	}

	sendCount := 1
	if latest != nil {
		if now.Sub(latest.LastSentAt) < time.Duration(s.cfg.OTPResendMinSeconds)*time.Second {
			return nil, &ServiceError{
				Status:  http.StatusTooManyRequests,
				Code:    "otp_resend_too_soon",
				Message: "OTP was sent too recently, please wait before retrying",
			}
		}

		sendCount = latest.SendCount
		if now.Sub(latest.LastSentAt) > time.Hour {
			sendCount = 0
		}
		if sendCount >= s.cfg.OTPMaxSendPerHour {
			return nil, &ServiceError{
				Status:  http.StatusTooManyRequests,
				Code:    "otp_rate_limited",
				Message: "Too many OTP requests for this email",
			}
		}
		sendCount++
	}

	code, err := s.generateCode()
	if err != nil {
		return nil, err
	}

	expiresAt := now.Add(time.Duration(s.cfg.OTPTTLSeconds) * time.Second)
	codeHash := HashCode(email, code, s.cfg.OTPSecret)

	otpID, err := s.storage.CreateOrReplace(ctx, email, codeHash, expiresAt, now, s.cfg.OTPMaxAttempts)
	if err != nil {
		return nil, err
	}
	if err := s.storage.UpdateResendMeta(ctx, otpID, now, sendCount); err != nil {
		return nil, err
	}

	subject := "Your HealthHub code"
	body := fmt.Sprintf("Код: %s. Действует %d минут.", code, s.cfg.OTPTTLSeconds/60)
	if err := s.sender.Send(email, subject, body); err != nil {
		_ = s.storage.MarkUsedOrDelete(ctx, otpID)
		return nil, err
	}

	resp := &RequestResponse{Status: "ok"}
	if s.cfg.Env == "local" && s.cfg.OTPDebugReturnCode {
		resp.DebugCode = &code
	}
	return resp, nil
}

func (s *Service) Verify(ctx context.Context, emailRaw, codeRaw string) (*VerifyResponse, error) {
	if !s.cfg.EmailAuthEnabled {
		return nil, &ServiceError{
			Status:  http.StatusNotFound,
			Code:    "email_auth_disabled",
			Message: "Email auth is disabled",
		}
	}
	if s.storage == nil {
		return nil, errors.New("email otp storage is not initialized")
	}

	email := normalizeEmail(emailRaw)
	code := strings.TrimSpace(codeRaw)
	if !isValidEmail(email) {
		return nil, &ServiceError{
			Status:  http.StatusBadRequest,
			Code:    "invalid_email",
			Message: "Invalid email format",
		}
	}
	if !isSixDigitCode(code) {
		return nil, &ServiceError{
			Status:  http.StatusBadRequest,
			Code:    "invalid_code_format",
			Message: "Code must contain exactly 6 digits",
		}
	}

	now := s.now()
	otp, err := s.storage.GetLatestActive(ctx, email, now)
	if err != nil {
		return nil, err
	}
	if otp == nil {
		return nil, &ServiceError{
			Status:  http.StatusUnauthorized,
			Code:    "otp_expired_or_not_found",
			Message: "OTP not found or expired",
		}
	}

	maxAttempts := otp.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = s.cfg.OTPMaxAttempts
	}
	if otp.Attempts >= maxAttempts {
		return nil, &ServiceError{
			Status:  http.StatusUnauthorized,
			Code:    "otp_locked",
			Message: "OTP is locked due to too many failed attempts",
		}
	}

	expectedHash := HashCode(email, code, s.cfg.OTPSecret)
	if !hmac.Equal([]byte(otp.CodeHash), []byte(expectedHash)) {
		if err := s.storage.IncrementAttempts(ctx, otp.ID); err != nil {
			return nil, err
		}
		return nil, &ServiceError{
			Status:  http.StatusUnauthorized,
			Code:    "otp_invalid_code",
			Message: "Invalid OTP code",
		}
	}

	if err := s.storage.MarkUsedOrDelete(ctx, otp.ID); err != nil {
		return nil, err
	}

	userID := "email:" + email
	token, expiresIn, err := s.generateAccessToken(userID)
	if err != nil {
		return nil, err
	}

	return &VerifyResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		UserID:      userID,
	}, nil
}

func (s *Service) generateAccessToken(userID string) (string, int64, error) {
	now := s.now()
	exp := now.Add(defaultTokenTTL)

	claims := jwt.MapClaims{
		"sub": userID,
		"iss": s.cfg.JWTIssuer,
		"exp": exp.Unix(),
		"iat": now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", 0, err
	}

	return signed, int64(defaultTokenTTL.Seconds()), nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isValidEmail(email string) bool {
	if email == "" || strings.Contains(email, " ") {
		return false
	}
	at := strings.Index(email, "@")
	dot := strings.LastIndex(email, ".")
	return at > 0 && dot > at+1 && dot < len(email)-1
}

func isSixDigitCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	_, err := strconv.Atoi(code)
	return err == nil
}

func GenerateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func HashCode(email, code, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strings.ToLower(strings.TrimSpace(email))))
	mac.Write([]byte(":"))
	mac.Write([]byte(strings.TrimSpace(code)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
