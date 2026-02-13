package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fdg312/health-hub/internal/config"
)

func TestCORS_PreflightAllowedOrigin(t *testing.T) {
	cfg := &config.Config{
		CORSAllowedOrigins: []string{"https://app.example.com"},
	}

	handler := CORSMiddleware(cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called for preflight")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/v1/profiles", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("expected Allow-Origin=https://app.example.com, got %q", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("expected Allow-Methods header to be set")
	}
	if got := rr.Header().Get("Access-Control-Max-Age"); got != "600" {
		t.Errorf("expected Max-Age=600, got %q", got)
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	cfg := &config.Config{
		CORSAllowedOrigins: []string{"https://app.example.com"},
	}

	innerCalled := false
	handler := CORSMiddleware(cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	// Normal request with disallowed origin
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles", nil)
	req.Header.Set("Origin", "https://evil.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !innerCalled {
		t.Error("expected inner handler to be called for non-OPTIONS request")
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no Allow-Origin header, got %q", got)
	}
}

func TestCORS_AllowedOriginOnNormalRequest(t *testing.T) {
	cfg := &config.Config{
		CORSAllowedOrigins:   []string{"https://app.example.com"},
		CORSAllowCredentials: true,
	}

	handler := CORSMiddleware(cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/profiles", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("expected Allow-Origin, got %q", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected Allow-Credentials=true, got %q", got)
	}
}

func TestCORS_NoOriginHeader(t *testing.T) {
	cfg := &config.Config{
		CORSAllowedOrigins: []string{"https://app.example.com"},
	}

	innerCalled := false
	handler := CORSMiddleware(cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/profiles", nil)
	// No Origin header
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !innerCalled {
		t.Error("expected inner handler to be called")
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no Allow-Origin header, got %q", got)
	}
}
