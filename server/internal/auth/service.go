package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/fdg312/health-hub/internal/config"
	"github.com/fdg312/health-hub/internal/storage"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken    = errors.New("invalid token")
	ErrTokenExpired    = errors.New("token expired")
	ErrInvalidAudience = errors.New("invalid audience")
	ErrProfileNotFound = errors.New("profile not found")
)

// AppleTokenVerifier — интерфейс для проверки Apple identity token
type AppleTokenVerifier interface {
	Verify(ctx context.Context, token string) (*AppleTokenClaims, error)
}

// Service — сервис авторизации
type Service struct {
	config        *config.Config
	storage       storage.Storage
	appleVerifier AppleTokenVerifier
	siwaService   *SIWAService
}

func NewService(cfg *config.Config, storage storage.Storage, appleVerifier AppleTokenVerifier) *Service {
	return NewServiceWithSIWA(cfg, storage, appleVerifier, nil)
}

func NewServiceWithSIWA(cfg *config.Config, storage storage.Storage, appleVerifier AppleTokenVerifier, siwaService *SIWAService) *Service {
	if siwaService == nil {
		siwaService = NewSIWAService(cfg, nil)
	}
	return &Service{
		config:        cfg,
		storage:       storage,
		appleVerifier: appleVerifier,
		siwaService:   siwaService,
	}
}

// SignInWithApple — авторизация через Apple
func (s *Service) SignInWithApple(ctx context.Context, req *SignInAppleRequest) (*SignInAppleResponse, error) {
	// 1. Verify Apple identity token
	claims, err := s.appleVerifier.Verify(ctx, req.IdentityToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify Apple token: %w", err)
	}

	ownerUserID := claims.Sub

	// 2. Find or create owner profile
	profile, err := s.findOrCreateOwnerProfile(ctx, ownerUserID, claims.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create owner profile: %w", err)
	}

	// 3. Generate JWT access token
	accessToken, err := s.generateJWT(ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	return &SignInAppleResponse{
		AccessToken:    accessToken,
		OwnerUserID:    ownerUserID,
		OwnerProfileID: profile.ID,
	}, nil
}

// SignInDev — dev-авторизация без Apple, выдает JWT на 30 дней
func (s *Service) SignInDev(ctx context.Context) (*DevAuthResponse, error) {
	_ = ctx

	const devUserID = "dev-user"
	const devTTL = 30 * 24 * time.Hour

	accessToken, err := s.generateJWTWithTTL(devUserID, devTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate dev JWT: %w", err)
	}

	return &DevAuthResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(devTTL.Seconds()),
	}, nil
}

// SignInSIWA verifies Apple identity token and returns our access token.
func (s *Service) SignInSIWA(ctx context.Context, req *SIWAAuthRequest) (*SIWAAuthResponse, error) {
	if strings.TrimSpace(req.IdentityToken) == "" {
		return nil, ErrInvalidIdentityToken
	}

	userID, _, err := s.siwaService.VerifyAppleIdentityToken(ctx, req.IdentityToken)
	if err != nil {
		return nil, err
	}

	const siwaTTL = 30 * 24 * time.Hour
	accessToken, err := s.generateJWTWithTTL(userID, siwaTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate siwa JWT: %w", err)
	}

	return &SIWAAuthResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(siwaTTL.Seconds()),
		UserID:      userID,
	}, nil
}

// findOrCreateOwnerProfile — найти или создать owner профиль
func (s *Service) findOrCreateOwnerProfile(ctx context.Context, ownerUserID, email string) (*storage.Profile, error) {
	// Try to find existing owner profile
	profiles, err := s.storage.ListProfiles(ctx)
	if err != nil {
		return nil, err
	}

	for _, p := range profiles {
		if p.Type == "owner" && p.OwnerUserID != "" && p.OwnerUserID == ownerUserID {
			return &p, nil
		}
	}

	// Create new owner profile
	name := "Я"
	if email != "" {
		name = email
	}

	profile := &storage.Profile{
		ID:          uuid.New(),
		Type:        "owner",
		Name:        name,
		OwnerUserID: ownerUserID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.storage.CreateProfile(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// generateJWT — генерация JWT токена
func (s *Service) generateJWT(ownerUserID string) (string, error) {
	return s.generateJWTWithTTL(ownerUserID, time.Duration(s.config.JWTTTLMinutes)*time.Minute)
}

func (s *Service) generateJWTWithTTL(ownerUserID string, ttl time.Duration) (string, error) {
	now := time.Now()
	exp := now.Add(ttl)

	claims := jwt.MapClaims{
		"sub": ownerUserID,
		"iss": s.config.JWTIssuer,
		"exp": exp.Unix(),
		"iat": now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

// VerifyJWT — проверка JWT токена
func (s *Service) VerifyJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return "", ErrInvalidToken
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		sub, ok := claims["sub"].(string)
		if !ok {
			return "", ErrInvalidToken
		}
		return sub, nil
	}

	return "", ErrInvalidToken
}

// RealAppleTokenVerifier — реальная реализация проверки Apple токенов
type RealAppleTokenVerifier struct {
	config *config.Config
	client *http.Client
}

func NewRealAppleTokenVerifier(cfg *config.Config) *RealAppleTokenVerifier {
	return &RealAppleTokenVerifier{
		config: cfg,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Verify — проверка Apple identity token
func (v *RealAppleTokenVerifier) Verify(ctx context.Context, tokenString string) (*AppleTokenClaims, error) {
	// Parse token without verification first to get header
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Get kid from header
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("missing kid in token header")
	}

	// Fetch Apple's public keys
	publicKey, err := v.fetchApplePublicKey(ctx, kid)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Apple public key: %w", err)
	}

	// Verify token signature
	token, err = jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to verify token signature: %w", err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	// Validate claims
	iss, _ := claims["iss"].(string)
	if iss != "https://appleid.apple.com" {
		return nil, errors.New("invalid issuer")
	}

	aud, _ := claims["aud"].(string)
	if v.config.AppleBundleID != "" && aud != v.config.AppleBundleID {
		return nil, ErrInvalidAudience
	}

	exp, _ := claims["exp"].(float64)
	if int64(exp) < time.Now().Unix() {
		return nil, ErrTokenExpired
	}

	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)

	if sub == "" {
		return nil, errors.New("missing sub claim")
	}

	return &AppleTokenClaims{
		Sub:   sub,
		Email: email,
		Aud:   aud,
		Exp:   int64(exp),
		Iat:   int64(claims["iat"].(float64)),
		Iss:   iss,
	}, nil
}

// fetchApplePublicKey — получение публичного ключа Apple
func (v *RealAppleTokenVerifier) fetchApplePublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://appleid.apple.com/auth/keys", nil)
	if err != nil {
		return nil, err
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Use string `json:"use"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}

	// Find key with matching kid
	for _, key := range jwks.Keys {
		if key.Kid == kid {
			return parseRSAPublicKey(key.N, key.E)
		}
	}

	return nil, fmt.Errorf("public key not found for kid: %s", kid)
}

// parseRSAPublicKey — парсинг RSA публичного ключа из JWK
func parseRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, err
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, err
	}

	n := new(big.Int).SetBytes(nBytes)

	var e int
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}

// MockAppleTokenVerifier — mock для тестов
type MockAppleTokenVerifier struct {
	VerifyFunc func(ctx context.Context, token string) (*AppleTokenClaims, error)
}

func (m *MockAppleTokenVerifier) Verify(ctx context.Context, token string) (*AppleTokenClaims, error) {
	if m.VerifyFunc != nil {
		return m.VerifyFunc(ctx, token)
	}

	// Default mock behavior for tests
	return &AppleTokenClaims{
		Sub:   strings.TrimPrefix(token, "mock_token_"),
		Email: "test@example.com",
		Aud:   "com.example.HealthHub",
		Exp:   time.Now().Add(time.Hour).Unix(),
		Iat:   time.Now().Unix(),
		Iss:   "https://appleid.apple.com",
	}, nil
}
