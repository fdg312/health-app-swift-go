package httpserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/fdg312/health-hub/internal/config"
)

// Server представляет HTTP сервер
type Server struct {
	config *config.Config
	mux    *http.ServeMux
}

// New создаёт новый HTTP сервер
func New(cfg *config.Config) *Server {
	s := &Server{
		config: cfg,
		mux:    http.NewServeMux(),
	}

	s.routes()
	return s
}

// routes регистрирует маршруты
func (s *Server) routes() {
	s.mux.HandleFunc("/healthz", s.handleHealthz)
}

// handleHealthz возвращает статус сервера
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// Start запускает HTTP сервер
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Port)

	if s.config.DatabaseURL == "" {
		log.Println("DATABASE_URL не указан, БД отключена")
	} else {
		log.Println("DATABASE_URL установлен (БД пока не используется)")
	}

	log.Printf("Сервер запущен на http://localhost%s\n", addr)
	log.Printf("Health check: http://localhost%s/healthz\n", addr)

	return http.ListenAndServe(addr, s.mux)
}
