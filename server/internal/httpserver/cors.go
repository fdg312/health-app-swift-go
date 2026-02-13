package httpserver

import (
	"net/http"
	"strings"

	"github.com/fdg312/health-hub/internal/config"
)

// CORSMiddleware returns an http.Handler that adds CORS headers.
func CORSMiddleware(cfg *config.Config, next http.Handler) http.Handler {
	allowed := make(map[string]bool, len(cfg.CORSAllowedOrigins))
	for _, o := range cfg.CORSAllowedOrigins {
		allowed[strings.TrimSpace(o)] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		if origin != "" && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")

			if cfg.CORSAllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}

		// Handle preflight OPTIONS
		if r.Method == http.MethodOptions && origin != "" {
			if allowed[origin] {
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type")
				w.Header().Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			// Origin not allowed — return 204 without CORS headers (browser will block)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
