package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fdg312/health-hub/internal/config"
	"github.com/fdg312/health-hub/internal/storage/memory"
	"github.com/google/uuid"
)

func setupTestService(authEnabled bool) (*Service, uuid.UUID) {
	memStorage := memory.New()
	cfg := &config.Config{
		AuthEnabled:    authEnabled,
		JWTSecret:      "test-secret-key-for-testing-only",
		JWTIssuer:      "health-hub-test",
		JWTTTLMinutes:  60,
		AppleBundleID:  "com.example.HealthHub",
		AppleIssuer:    "https://appleid.apple.com",
		AppleJWKSURL:   "http://localhost/does-not-matter",
		AppleSubPrefix: "apple:",
	}

	mockVerifier := &MockAppleTokenVerifier{}
	service := NewService(cfg, memStorage, mockVerifier)

	ctx := context.Background()
	profiles, _ := memStorage.ListProfiles(ctx)
	ownerID := profiles[0].ID

	return service, ownerID
}

func TestHandleSignInApple(t *testing.T) {
	service, _ := setupTestService(true)
	handler := NewHandlers(service)

	t.Run("Success", func(t *testing.T) {
		reqBody := SignInAppleRequest{
			IdentityToken: "mock_token_test_user_123",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/v1/auth/apple", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.HandleSignInApple(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp SignInAppleResponse
		json.NewDecoder(w.Body).Decode(&resp)

		if resp.AccessToken == "" {
			t.Error("expected access_token not empty")
		}

		if resp.OwnerUserID != "test_user_123" {
			t.Errorf("expected owner_user_id 'test_user_123', got '%s'", resp.OwnerUserID)
		}

		if resp.OwnerProfileID == uuid.Nil {
			t.Error("expected owner_profile_id not nil")
		}
	})

	t.Run("MissingToken", func(t *testing.T) {
		reqBody := SignInAppleRequest{}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/v1/auth/apple", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.HandleSignInApple(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestHandleDevAuth(t *testing.T) {
	service, _ := setupTestService(true)
	handler := NewHandlers(service)

	req := httptest.NewRequest("POST", "/v1/auth/dev", nil)
	w := httptest.NewRecorder()

	handler.HandleDevAuth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp DevAuthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("expected access_token not empty")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("expected token_type Bearer, got %q", resp.TokenType)
	}
	if resp.ExpiresIn != int64((30 * 24 * time.Hour).Seconds()) {
		t.Errorf("expected expires_in 2592000, got %d", resp.ExpiresIn)
	}
}

func TestMiddlewareAuth(t *testing.T) {
	service, _ := setupTestService(true)
	cfg := &config.Config{
		AuthEnabled:   true,
		AuthRequired:  true,
		JWTSecret:     "test-secret-key-for-testing-only",
		JWTIssuer:     "health-hub-test",
		JWTTTLMinutes: 60,
	}

	middleware := NewMiddleware(cfg, service)

	t.Run("ValidToken", func(t *testing.T) {
		// Generate valid JWT
		token, err := service.generateJWT("test_user_123")
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest("GET", "/v1/profiles", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		var calledNext bool
		handler := middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calledNext = true
			ownerUserID, ok := GetOwnerUserID(r.Context())
			if !ok || ownerUserID != "test_user_123" {
				t.Errorf("expected owner_user_id in context")
			}
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(w, req)

		if !calledNext {
			t.Error("expected next handler to be called")
		}

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("MissingToken", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/profiles", nil)
		w := httptest.NewRecorder()

		handler := middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("should not call next handler")
		}))

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("InvalidToken", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/profiles", nil)
		req.Header.Set("Authorization", "Bearer invalid_token")
		w := httptest.NewRecorder()

		handler := middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("should not call next handler")
		}))

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})
}

func TestMiddlewareAuthDisabled(t *testing.T) {
	service, _ := setupTestService(false)
	cfg := &config.Config{
		AuthEnabled:  false,
		AuthRequired: false,
	}

	middleware := NewMiddleware(cfg, service)

	req := httptest.NewRequest("GET", "/v1/profiles", nil)
	w := httptest.NewRecorder()

	var calledNext bool
	handler := middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledNext = true
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)

	if !calledNext {
		t.Error("expected next handler to be called when auth disabled")
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestOptionalAuthMiddleware(t *testing.T) {
	service, _ := setupTestService(true)
	cfg := &config.Config{
		JWTSecret:     "test-secret-key-for-testing-only",
		JWTIssuer:     "health-hub-test",
		JWTTTLMinutes: 60,
	}

	middleware := NewMiddleware(cfg, service)

	t.Run("NoTokenPasses", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/profiles", nil)
		w := httptest.NewRecorder()

		var called bool
		handler := middleware.OptionalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(w, req)

		if !called || w.Code != http.StatusOK {
			t.Fatalf("expected passthrough with 200, got called=%v status=%d", called, w.Code)
		}
	})

	t.Run("ValidTokenAddsContext", func(t *testing.T) {
		token, err := service.generateJWT("test_user_123")
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest("GET", "/v1/profiles", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		var gotSub string
		handler := middleware.OptionalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sub, _ := GetOwnerUserID(r.Context())
			gotSub = sub
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if gotSub != "test_user_123" {
			t.Fatalf("expected sub in context, got %q", gotSub)
		}
	})

	t.Run("InvalidTokenRejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/profiles", nil)
		req.Header.Set("Authorization", "Bearer invalid")
		w := httptest.NewRecorder()

		handler := middleware.OptionalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("should not call next handler")
		}))

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("DevAuthPathAlwaysAccessible", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/v1/auth/dev", nil)
		req.Header.Set("Authorization", "Bearer invalid")
		w := httptest.NewRecorder()

		var called bool
		handler := middleware.OptionalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(w, req)

		if !called || w.Code != http.StatusOK {
			t.Fatalf("expected /v1/auth/dev passthrough, called=%v status=%d", called, w.Code)
		}
	})
}

func TestJWTGeneration(t *testing.T) {
	service, _ := setupTestService(true)

	token, err := service.generateJWT("test_user_123")
	if err != nil {
		t.Fatal(err)
	}

	if token == "" {
		t.Error("expected token not empty")
	}

	// Verify token
	ownerUserID, err := service.VerifyJWT(token)
	if err != nil {
		t.Fatal(err)
	}

	if ownerUserID != "test_user_123" {
		t.Errorf("expected owner_user_id 'test_user_123', got '%s'", ownerUserID)
	}
}

func TestFindOrCreateOwnerProfile(t *testing.T) {
	service, existingOwnerID := setupTestService(true)
	ctx := context.Background()

	t.Run("CreateNewOwner", func(t *testing.T) {
		profile, err := service.findOrCreateOwnerProfile(ctx, "new_user_456", "new@example.com")
		if err != nil {
			t.Fatal(err)
		}

		if profile.Type != "owner" {
			t.Errorf("expected type 'owner', got '%s'", profile.Type)
		}

		if profile.OwnerUserID == "" || profile.OwnerUserID != "new_user_456" {
			t.Error("expected owner_user_id set correctly")
		}
	})

	t.Run("FindExistingOwner", func(t *testing.T) {
		// Create first profile
		profile1, err := service.findOrCreateOwnerProfile(ctx, "existing_user_789", "existing@example.com")
		if err != nil {
			t.Fatal(err)
		}

		// Try to create again - should find existing
		profile2, err := service.findOrCreateOwnerProfile(ctx, "existing_user_789", "existing@example.com")
		if err != nil {
			t.Fatal(err)
		}

		if profile1.ID != profile2.ID {
			t.Error("expected same profile ID for existing user")
		}
	})

	_ = existingOwnerID // Keep initial owner
}
