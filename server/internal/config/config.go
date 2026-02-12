package config

import (
	"os"
	"strconv"
)

// Config содержит конфигурацию приложения
type Config struct {
	Port        int
	DatabaseURL string
}

// Load загружает конфигурацию из переменных окружения
func Load() *Config {
	port := 8080
	if portStr := os.Getenv("PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	return &Config{
		Port:        port,
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
}
