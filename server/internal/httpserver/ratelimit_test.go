package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fdg312/health-hub/internal/config"
)

func TestRateLimit_SecondRequestReturns429(t *testing.T) {
	cfg := &config.Config{
		RateLimitRPS:   1,
		RateLimitBurst: 1,
	}

	handler := RateLimitMiddleware(cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request should succeed
	req1 := httptest.NewRequest(http.MethodGet, "/v1/profiles", nil)
	req1.RemoteAddr = "1.2.3.4:12345"
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", rr1.Code)
	}

	// Second immediate request should be rate-limited
	req2 := httptest.NewRequest(http.MethodGet, "/v1/profiles", nil)
	req2.RemoteAddr = "1.2.3.4:12345"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: expected 429, got %d", rr2.Code)
	}

	// Verify response body
	var body map[string]interface{}
	if err := json.NewDecoder(rr2.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	errObj, ok := body["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object in response")
	}
	if errObj["code"] != "rate_limited" {
		t.Errorf("expected code=rate_limited, got %v", errObj["code"])
	}
}

func TestRateLimit_DisabledWhenZero(t *testing.T) {
	cfg := &config.Config{
		RateLimitRPS:   0,
		RateLimitBurst: 0,
	}

	callCount := 0
	handler := RateLimitMiddleware(cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))

	// Multiple rapid requests should all succeed
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, rr.Code)
		}
	}

	if callCount != 10 {
		t.Errorf("expected 10 calls, got %d", callCount)
	}
}

func TestRateLimit_DifferentIPsIndependent(t *testing.T) {
	cfg := &config.Config{
		RateLimitRPS:   1,
		RateLimitBurst: 1,
	}

	handler := RateLimitMiddleware(cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First IP uses its token
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "1.2.3.4:1"
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("IP1 first request: expected 200, got %d", rr1.Code)
	}

	// Second IP should still succeed (independent bucket)
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "5.6.7.8:1"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("IP2 first request: expected 200, got %d", rr2.Code)
	}
}
