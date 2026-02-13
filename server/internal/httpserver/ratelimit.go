package httpserver

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/fdg312/health-hub/internal/config"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter *rate.Limiter
}

type rateLimiterStore struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	rps      rate.Limit
	burst    int
	counter  atomic.Int64
}

func newRateLimiterStore(rps int, burst int) *rateLimiterStore {
	return &rateLimiterStore{
		limiters: make(map[string]*ipLimiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

func (s *rateLimiterStore) getLimiter(ip string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.limiters[ip]
	if !exists {
		entry = &ipLimiter{
			limiter: rate.NewLimiter(s.rps, s.burst),
		}
		s.limiters[ip] = entry
	}

	// Simple cleanup: every 1000 requests, clear old entries to prevent unbounded growth.
	count := s.counter.Add(1)
	if count%1000 == 0 {
		s.cleanup()
	}

	return entry.limiter
}

// cleanup removes IPs whose token bucket is full (idle clients).
func (s *rateLimiterStore) cleanup() {
	for ip, entry := range s.limiters {
		// If the limiter has full tokens, the client has been idle — safe to evict.
		if entry.limiter.Tokens() >= float64(s.burst) {
			delete(s.limiters, ip)
		}
	}
}

// RateLimitMiddleware enforces per-IP rate limiting via token bucket.
// If RateLimitRPS <= 0, the middleware is a no-op pass-through.
func RateLimitMiddleware(cfg *config.Config, next http.Handler) http.Handler {
	if cfg.RateLimitRPS <= 0 {
		return next // disabled
	}

	burst := cfg.RateLimitBurst
	if burst <= 0 {
		burst = cfg.RateLimitRPS
	}

	store := newRateLimiterStore(cfg.RateLimitRPS, burst)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		limiter := store.getLimiter(ip)

		if !limiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"code":    "rate_limited",
					"message": "Too many requests",
				},
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func extractIP(r *http.Request) string {
	// Prefer X-Forwarded-For for proxied setups
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take first IP in the chain
		if idx := len(xff); idx > 0 {
			for i, ch := range xff {
				if ch == ',' {
					return xff[:i]
				}
			}
			return xff
		}
	}

	// Fallback to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
