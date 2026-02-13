package ai

import (
	"strings"

	"github.com/fdg312/health-hub/internal/config"
)

const (
	ModeMock   = "mock"
	ModeOpenAI = "openai"
)

func NewProvider(cfg *config.Config) Provider {
	mode := strings.ToLower(strings.TrimSpace(cfg.AIMode))
	if mode == "" {
		mode = ModeMock
	}

	switch mode {
	case ModeOpenAI:
		return NewOpenAIProvider(cfg)
	default:
		return NewMockProvider()
	}
}
