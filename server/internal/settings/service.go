package settings

import (
	"context"
	"fmt"
	"strings"

	"github.com/fdg312/health-hub/internal/config"
	"github.com/fdg312/health-hub/internal/storage"
)

const (
	defaultMorningCheckinMinutes = 540  // 09:00
	defaultEveningCheckinMinutes = 1260 // 21:00
	defaultVitaminsTimeMinutes   = 720  // 12:00
)

type Service struct {
	storage storage.SettingsStorage
	config  *config.Config
}

func NewService(settingsStorage storage.SettingsStorage, cfg *config.Config) *Service {
	return &Service{
		storage: settingsStorage,
		config:  cfg,
	}
}

func (s *Service) GetOrDefault(ctx context.Context, ownerUserID string) (SettingsResponse, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return SettingsResponse{}, fmt.Errorf("owner_user_id is required")
	}

	row, found, err := s.storage.GetSettings(ctx, ownerUserID)
	if err != nil {
		return SettingsResponse{}, err
	}

	if !found {
		return SettingsResponse{
			Settings:  dtoFromStorage(s.defaults()),
			IsDefault: true,
		}, nil
	}

	return SettingsResponse{
		Settings:  dtoFromStorage(row),
		IsDefault: false,
	}, nil
}

func (s *Service) Upsert(ctx context.Context, ownerUserID string, dto SettingsDTO) (SettingsDTO, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return SettingsDTO{}, fmt.Errorf("owner_user_id is required")
	}

	if err := dto.Validate(); err != nil {
		return SettingsDTO{}, err
	}

	row, err := s.storage.UpsertSettings(ctx, ownerUserID, dtoToStorage(dto))
	if err != nil {
		return SettingsDTO{}, err
	}
	return dtoFromStorage(row), nil
}

func (s *Service) defaults() storage.Settings {
	return storage.Settings{
		NotificationsMaxPerDay: s.config.NotificationsMaxPerDay,
		MinSleepMinutes:        s.config.DefaultSleepMinMinutes,
		MinSteps:               s.config.DefaultStepsMin,
		MinActiveEnergyKcal:    s.config.DefaultActiveEnergyMinKcal,
		MorningCheckinMinute:   defaultMorningCheckinMinutes,
		EveningCheckinMinute:   defaultEveningCheckinMinutes,
		VitaminsTimeMinute:     defaultVitaminsTimeMinutes,
	}
}
