package auth

import (
	"log"
	"net/http"
	"strings"

	"github.com/fdg312/health-hub/internal/config"
)

// Middleware — middleware для проверки авторизации
type Middleware struct {
	config  *config.Config
	service *Service
}

func NewMiddleware(cfg *config.Config, service *Service) *Middleware {
	return &Middleware{
		config:  cfg,
		service: service,
	}
}

// RequireAuth — middleware для защиты эндпоинтов
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.config.AuthRequired || isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		userID, err := m.authenticateHeader(r.Header.Get("Authorization"))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
			return
		}

		next.ServeHTTP(w, r.WithContext(WithUserID(r.Context(), userID)))
	})
}

// OptionalAuth validates Bearer token only when it is provided.
// Without token, requests pass through unchanged.
func (m *Middleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Public endpoints must always be reachable.
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if strings.TrimSpace(authHeader) == "" {
			next.ServeHTTP(w, r)
			return
		}

		userID, err := m.authenticateHeader(authHeader)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
			return
		}

		log.Printf("auth token accepted: sub=%s method=%s path=%s", userID, r.Method, r.URL.Path)
		next.ServeHTTP(w, r.WithContext(WithUserID(r.Context(), userID)))
	})
}

func (m *Middleware) authenticateHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", ErrInvalidToken
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", ErrInvalidToken
	}

	return m.service.VerifyJWT(parts[1])
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":{"code":"` + code + `","message":"` + message + `"}}`))
}

func isPublicPath(path string) bool {
	return path == "/healthz" || strings.HasPrefix(path, "/v1/auth/")
}
