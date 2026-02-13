package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fdg312/health-hub/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestHealthz(t *testing.T) {
	cfg := &config.Config{Port: 8080}
	srv := New(cfg)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", resp["status"])
	}
}

func TestHealthzMethodNotAllowed(t *testing.T) {
	cfg := &config.Config{Port: 8080}
	srv := New(cfg)

	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestAuthRequiredWithoutToken(t *testing.T) {
	cfg := &config.Config{
		Port:         8080,
		AuthMode:     "dev",
		AuthEnabled:  true,
		AuthRequired: true,
		JWTSecret:    "test-secret",
		JWTIssuer:    "health-hub-test",
	}
	srv := New(cfg)

	var handler http.Handler = srv.mux
	if srv.authMiddleware != nil {
		handler = srv.authMiddleware.RequireAuth(handler)
	}
	handler = RateLimitMiddleware(cfg, handler)
	handler = CORSMiddleware(cfg, handler)

	req := httptest.NewRequest(http.MethodGet, "/v1/profiles", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestUserIsolationAcrossTokens(t *testing.T) {
	cfg := &config.Config{
		Port:         8080,
		AuthMode:     "dev",
		AuthEnabled:  true,
		AuthRequired: true,
		JWTSecret:    "test-secret",
		JWTIssuer:    "health-hub-test",
	}
	srv := New(cfg)
	handler := buildServerHandler(srv, cfg)

	tokenA := testJWT(t, cfg.JWTSecret, cfg.JWTIssuer, "userA")
	tokenB := testJWT(t, cfg.JWTSecret, cfg.JWTIssuer, "userB")

	ownerA := listOwnerProfileID(t, handler, tokenA)
	ownerB := listOwnerProfileID(t, handler, tokenB)
	if ownerA == ownerB {
		t.Fatalf("expected different owner profiles for userA and userB")
	}

	// Create guest under userA
	guestA := createGuestProfile(t, handler, tokenA, "Guest A")

	// userB should not see userA guest profile
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles", nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var listResp struct {
		Profiles []struct {
			ID uuid.UUID `json:"id"`
		} `json:"profiles"`
	}
	if err := json.NewDecoder(w.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode profiles response: %v", err)
	}
	for _, p := range listResp.Profiles {
		if p.ID == guestA {
			t.Fatalf("userB unexpectedly sees userA profile %s", guestA)
		}
	}

	// userA can create checkin on guestA
	checkinBody := []byte(`{"profile_id":"` + guestA.String() + `","date":"2026-02-13","type":"morning","score":4}`)
	req = httptest.NewRequest(http.MethodPost, "/v1/checkins", bytes.NewReader(checkinBody))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 creating checkin, got %d body=%s", w.Code, w.Body.String())
	}

	// userB must get 404 when reading userA checkins
	req = httptest.NewRequest(http.MethodGet, "/v1/checkins?profile_id="+guestA.String()+"&from=2026-02-13&to=2026-02-13", nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for cross-user checkins, got %d body=%s", w.Code, w.Body.String())
	}
}

func buildServerHandler(srv *Server, cfg *config.Config) http.Handler {
	var handler http.Handler = srv.mux
	if srv.authMiddleware != nil && cfg.AuthMode != "none" {
		if cfg.AuthRequired {
			handler = srv.authMiddleware.RequireAuth(handler)
		} else {
			handler = srv.authMiddleware.OptionalAuth(handler)
		}
	}
	handler = RateLimitMiddleware(cfg, handler)
	handler = CORSMiddleware(cfg, handler)
	return handler
}

func testJWT(t *testing.T, secret, issuer, sub string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": sub,
		"iss": issuer,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}
	return tokenStr
}

func listOwnerProfileID(t *testing.T, handler http.Handler, token string) uuid.UUID {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Profiles []struct {
			ID   uuid.UUID `json:"id"`
			Type string    `json:"type"`
		} `json:"profiles"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode profiles: %v", err)
	}
	for _, p := range resp.Profiles {
		if p.Type == "owner" {
			return p.ID
		}
	}
	t.Fatalf("owner profile not found in response")
	return uuid.Nil
}

func createGuestProfile(t *testing.T, handler http.Handler, token, name string) uuid.UUID {
	t.Helper()
	body := []byte(`{"type":"guest","name":"` + name + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/profiles", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", w.Code, w.Body.String())
	}
	var profile struct {
		ID uuid.UUID `json:"id"`
	}
	if err := json.NewDecoder(w.Body).Decode(&profile); err != nil {
		t.Fatalf("decode guest: %v", err)
	}
	return profile.ID
}
