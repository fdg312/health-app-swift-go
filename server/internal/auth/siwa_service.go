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
	"sync"
	"time"

	"github.com/fdg312/health-hub/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidIdentityToken = errors.New("invalid identity token")
	ErrJWKSFetchFailed      = errors.New("jwks fetch failed")
)

type jwksDocument struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// JWKSClient fetches and caches Apple public keys.
type JWKSClient struct {
	url        string
	httpClient *http.Client
	cacheTTL   time.Duration

	mu      sync.RWMutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

func NewJWKSClient(url string, client *http.Client) *JWKSClient {
	jwksURL := strings.TrimSpace(url)
	if jwksURL == "" {
		jwksURL = "https://appleid.apple.com/auth/keys"
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &JWKSClient{
		url:        jwksURL,
		httpClient: client,
		cacheTTL:   time.Hour,
		keys:       make(map[string]*rsa.PublicKey),
	}
}

func (c *JWKSClient) GetKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	if strings.TrimSpace(kid) == "" {
		return nil, ErrInvalidIdentityToken
	}

	c.mu.RLock()
	isFresh := !c.fetched.IsZero() && time.Since(c.fetched) < c.cacheTTL
	key, exists := c.keys[kid]
	c.mu.RUnlock()

	if isFresh && exists {
		return key, nil
	}

	if err := c.refresh(ctx); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrJWKSFetchFailed, err)
	}

	c.mu.RLock()
	key, exists = c.keys[kid]
	c.mu.RUnlock()
	if !exists {
		return nil, ErrInvalidIdentityToken
	}
	return key, nil
}

func (c *JWKSClient) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected JWKS status: %d", resp.StatusCode)
	}

	var doc jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return err
	}

	newKeys := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, key := range doc.Keys {
		if strings.TrimSpace(key.Kid) == "" || strings.ToUpper(key.Kty) != "RSA" {
			continue
		}
		pubKey, err := parseRSAPublicKeyFromJWK(key.N, key.E)
		if err != nil {
			continue
		}
		newKeys[key.Kid] = pubKey
	}
	if len(newKeys) == 0 {
		return errors.New("jwks does not contain usable rsa keys")
	}

	c.mu.Lock()
	c.keys = newKeys
	c.fetched = time.Now()
	c.mu.Unlock()
	return nil
}

func parseRSAPublicKeyFromJWK(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, err
	}

	modulus := new(big.Int).SetBytes(nBytes)
	exponent := 0
	for _, b := range eBytes {
		exponent = exponent<<8 + int(b)
	}
	if exponent <= 0 {
		return nil, errors.New("invalid rsa exponent")
	}
	return &rsa.PublicKey{N: modulus, E: exponent}, nil
}

// SIWAService verifies Apple identity token and maps it to local user_id.
type SIWAService struct {
	config     *config.Config
	jwksClient *JWKSClient
	now        func() time.Time
}

func NewSIWAService(cfg *config.Config, jwksClient *JWKSClient) *SIWAService {
	client := jwksClient
	if client == nil {
		client = NewJWKSClient(cfg.AppleJWKSURL, nil)
	}
	return &SIWAService{
		config:     cfg,
		jwksClient: client,
		now:        time.Now,
	}
}

func (s *SIWAService) VerifyAppleIdentityToken(ctx context.Context, tokenString string) (string, *AppleIdentityClaims, error) {
	rawToken := strings.TrimSpace(tokenString)
	if rawToken == "" {
		return "", nil, ErrInvalidIdentityToken
	}

	parsedUnverified, _, err := new(jwt.Parser).ParseUnverified(rawToken, jwt.MapClaims{})
	if err != nil {
		return "", nil, ErrInvalidIdentityToken
	}
	kid, ok := parsedUnverified.Header["kid"].(string)
	if !ok || strings.TrimSpace(kid) == "" {
		return "", nil, ErrInvalidIdentityToken
	}

	publicKey, err := s.jwksClient.GetKey(ctx, kid)
	if err != nil {
		return "", nil, err
	}

	claims := &AppleIdentityClaims{}
	parserOptions := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithIssuer(s.config.AppleIssuer),
		jwt.WithTimeFunc(s.now),
	}
	if s.config.AppleBundleID != "" {
		parserOptions = append(parserOptions, jwt.WithAudience(s.config.AppleBundleID))
	}

	token, err := jwt.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	}, parserOptions...)
	if err != nil || !token.Valid {
		return "", nil, ErrInvalidIdentityToken
	}

	now := s.now()
	if claims.Subject == "" || claims.Issuer == "" || claims.ExpiresAt == nil || !claims.ExpiresAt.After(now) {
		return "", nil, ErrInvalidIdentityToken
	}
	if claims.IssuedAt != nil && claims.IssuedAt.Time.After(now.Add(5*time.Minute)) {
		return "", nil, ErrInvalidIdentityToken
	}

	userID := s.config.AppleSubPrefix + claims.Subject
	return userID, claims, nil
}
