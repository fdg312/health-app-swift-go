package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fdg312/health-hub/internal/config"
	"github.com/fdg312/health-hub/internal/storage/memory"
	"github.com/golang-jwt/jwt/v5"
)

func TestSIWAServiceVerifyAppleIdentityToken(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	const kid = "test-kid"

	jwksServer := newTestJWKSServer(t, privateKey, kid)
	defer jwksServer.Close()

	cfg := &config.Config{
		AppleBundleID:  "com.example.HealthHub",
		AppleIssuer:    "https://appleid.apple.com",
		AppleJWKSURL:   jwksServer.URL,
		AppleSubPrefix: "apple:",
	}
	service := NewSIWAService(cfg, NewJWKSClient(jwksServer.URL, jwksServer.Client()))

	t.Run("valid token", func(t *testing.T) {
		identityToken := signAppleIdentityToken(t, privateKey, kid, "https://appleid.apple.com", "com.example.HealthHub", "sub-123", time.Now().Add(10*time.Minute))
		userID, claims, err := service.VerifyAppleIdentityToken(context.Background(), identityToken)
		if err != nil {
			t.Fatalf("verify token: %v", err)
		}
		if userID != "apple:sub-123" {
			t.Fatalf("expected user_id apple:sub-123, got %q", userID)
		}
		if claims.Subject != "sub-123" {
			t.Fatalf("expected subject sub-123, got %q", claims.Subject)
		}
	})

	t.Run("invalid audience", func(t *testing.T) {
		identityToken := signAppleIdentityToken(t, privateKey, kid, "https://appleid.apple.com", "com.other.app", "sub-123", time.Now().Add(10*time.Minute))
		_, _, err := service.VerifyAppleIdentityToken(context.Background(), identityToken)
		if err == nil {
			t.Fatal("expected error")
		}
		if err != ErrInvalidIdentityToken {
			t.Fatalf("expected ErrInvalidIdentityToken, got %v", err)
		}
	})

	t.Run("invalid issuer", func(t *testing.T) {
		identityToken := signAppleIdentityToken(t, privateKey, kid, "https://issuer.invalid", "com.example.HealthHub", "sub-123", time.Now().Add(10*time.Minute))
		_, _, err := service.VerifyAppleIdentityToken(context.Background(), identityToken)
		if err == nil {
			t.Fatal("expected error")
		}
		if err != ErrInvalidIdentityToken {
			t.Fatalf("expected ErrInvalidIdentityToken, got %v", err)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		identityToken := signAppleIdentityToken(t, privateKey, kid, "https://appleid.apple.com", "com.example.HealthHub", "sub-123", time.Now().Add(-10*time.Minute))
		_, _, err := service.VerifyAppleIdentityToken(context.Background(), identityToken)
		if err == nil {
			t.Fatal("expected error")
		}
		if err != ErrInvalidIdentityToken {
			t.Fatalf("expected ErrInvalidIdentityToken, got %v", err)
		}
	})
}

func TestHandleSignInSIWA(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	const kid = "test-kid"

	jwksServer := newTestJWKSServer(t, privateKey, kid)
	defer jwksServer.Close()

	cfg := &config.Config{
		JWTSecret:      "test-secret",
		JWTIssuer:      "health-hub-test",
		AppleBundleID:  "com.example.HealthHub",
		AppleIssuer:    "https://appleid.apple.com",
		AppleJWKSURL:   jwksServer.URL,
		AppleSubPrefix: "apple:",
	}
	siwaService := NewSIWAService(cfg, NewJWKSClient(jwksServer.URL, jwksServer.Client()))
	authService := NewServiceWithSIWA(cfg, memory.New(), &MockAppleTokenVerifier{}, siwaService)
	handler := NewHandlers(authService)

	t.Run("success", func(t *testing.T) {
		identityToken := signAppleIdentityToken(t, privateKey, kid, "https://appleid.apple.com", "com.example.HealthHub", "user-001", time.Now().Add(10*time.Minute))
		body, _ := json.Marshal(SIWAAuthRequest{IdentityToken: identityToken})
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/siwa", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.HandleSignInSIWA(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
		}

		var resp SIWAAuthResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.AccessToken == "" {
			t.Fatal("expected access_token")
		}
		if resp.UserID != "apple:user-001" {
			t.Fatalf("expected user_id apple:user-001, got %q", resp.UserID)
		}
	})

	t.Run("jwks fetch failed", func(t *testing.T) {
		cfg := &config.Config{
			JWTSecret:      "test-secret",
			JWTIssuer:      "health-hub-test",
			AppleBundleID:  "com.example.HealthHub",
			AppleIssuer:    "https://appleid.apple.com",
			AppleJWKSURL:   "http://127.0.0.1:1",
			AppleSubPrefix: "apple:",
		}
		siwaService := NewSIWAService(cfg, NewJWKSClient(cfg.AppleJWKSURL, &http.Client{Timeout: 50 * time.Millisecond}))
		authService := NewServiceWithSIWA(cfg, memory.New(), &MockAppleTokenVerifier{}, siwaService)
		handler := NewHandlers(authService)

		body, _ := json.Marshal(SIWAAuthRequest{IdentityToken: "eyJhbGciOiJSUzI1NiIsImtpZCI6InRlc3QifQ.eyJzdWIiOiJ1c2VyIn0.sig"})
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/siwa", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.HandleSignInSIWA(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d body=%s", w.Code, w.Body.String())
		}
		var resp ErrorResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode error response: %v", err)
		}
		if resp.Error.Code != "jwks_fetch_failed" {
			t.Fatalf("expected jwks_fetch_failed, got %q", resp.Error.Code)
		}
	})
}

func newTestJWKSServer(t *testing.T, privateKey *rsa.PrivateKey, kid string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pub := privateKey.Public().(*rsa.PublicKey)
		jwks := map[string]any{
			"keys": []map[string]any{
				{
					"kid": kid,
					"kty": "RSA",
					"use": "sig",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(bigEndianBytes(pub.E)),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
}

func signAppleIdentityToken(t *testing.T, privateKey *rsa.PrivateKey, kid, iss, aud, sub string, exp time.Time) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":   iss,
		"aud":   aud,
		"sub":   sub,
		"email": "user@example.com",
		"iat":   time.Now().Add(-1 * time.Minute).Unix(),
		"exp":   exp.Unix(),
	})
	token.Header["kid"] = kid
	signed, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func bigEndianBytes(v int) []byte {
	if v == 0 {
		return []byte{0}
	}
	out := []byte{}
	for value := v; value > 0; value >>= 8 {
		out = append([]byte{byte(value & 0xff)}, out...)
	}
	return out
}
