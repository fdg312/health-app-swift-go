package settings

import (
	"fmt"
	"strings"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
)

type SettingsDTO struct {
	TimeZone *string `json:"time_zone,omitempty"`

	QuietStartMinutes *int `json:"quiet_start_minutes,omitempty"`
	QuietEndMinutes   *int `json:"quiet_end_minutes,omitempty"`

	NotificationsMaxPerDay int `json:"notifications_max_per_day"`

	MinSleepMinutes     int `json:"min_sleep_minutes"`
	MinSteps            int `json:"min_steps"`
	MinActiveEnergyKcal int `json:"min_active_energy_kcal"`

	MorningCheckinTimeMinutes int `json:"morning_checkin_time_minutes"`
	EveningCheckinTimeMinutes int `json:"evening_checkin_time_minutes"`
	VitaminsTimeMinutes       int `json:"vitamins_time_minutes"`
}

type SettingsResponse struct {
	Settings  SettingsDTO `json:"settings"`
	IsDefault bool        `json:"is_default"`
}

func (s SettingsDTO) Validate() error {
	if s.TimeZone != nil && strings.TrimSpace(*s.TimeZone) != "" {
		if _, err := time.LoadLocation(strings.TrimSpace(*s.TimeZone)); err != nil {
			return fmt.Errorf("invalid time_zone")
		}
	}

	if (s.QuietStartMinutes == nil) != (s.QuietEndMinutes == nil) {
		return fmt.Errorf("quiet_start_minutes and quiet_end_minutes must be both set or both null")
	}
	if s.QuietStartMinutes != nil && (*s.QuietStartMinutes < 0 || *s.QuietStartMinutes > 1439) {
		return fmt.Errorf("quiet_start_minutes must be in range 0..1439")
	}
	if s.QuietEndMinutes != nil && (*s.QuietEndMinutes < 0 || *s.QuietEndMinutes > 1439) {
		return fmt.Errorf("quiet_end_minutes must be in range 0..1439")
	}

	if s.NotificationsMaxPerDay < 0 || s.NotificationsMaxPerDay > 10 {
		return fmt.Errorf("notifications_max_per_day must be in range 0..10")
	}
	if s.MinSleepMinutes < 0 || s.MinSleepMinutes > 1200 {
		return fmt.Errorf("min_sleep_minutes must be in range 0..1200")
	}
	if s.MinSteps < 0 || s.MinSteps > 50000 {
		return fmt.Errorf("min_steps must be in range 0..50000")
	}
	if s.MinActiveEnergyKcal < 0 || s.MinActiveEnergyKcal > 5000 {
		return fmt.Errorf("min_active_energy_kcal must be in range 0..5000")
	}
	if s.MorningCheckinTimeMinutes < 0 || s.MorningCheckinTimeMinutes > 1439 {
		return fmt.Errorf("morning_checkin_time_minutes must be in range 0..1439")
	}
	if s.EveningCheckinTimeMinutes < 0 || s.EveningCheckinTimeMinutes > 1439 {
		return fmt.Errorf("evening_checkin_time_minutes must be in range 0..1439")
	}
	if s.VitaminsTimeMinutes < 0 || s.VitaminsTimeMinutes > 1439 {
		return fmt.Errorf("vitamins_time_minutes must be in range 0..1439")
	}

	return nil
}

func dtoFromStorage(s storage.Settings) SettingsDTO {
	return SettingsDTO{
		TimeZone:                  cloneStringPointer(s.TimeZone),
		QuietStartMinutes:         cloneIntPointer(s.QuietStartMinutes),
		QuietEndMinutes:           cloneIntPointer(s.QuietEndMinutes),
		NotificationsMaxPerDay:    s.NotificationsMaxPerDay,
		MinSleepMinutes:           s.MinSleepMinutes,
		MinSteps:                  s.MinSteps,
		MinActiveEnergyKcal:       s.MinActiveEnergyKcal,
		MorningCheckinTimeMinutes: s.MorningCheckinMinute,
		EveningCheckinTimeMinutes: s.EveningCheckinMinute,
		VitaminsTimeMinutes:       s.VitaminsTimeMinute,
	}
}

func dtoToStorage(dto SettingsDTO) storage.Settings {
	return storage.Settings{
		TimeZone:               cloneStringPointer(dto.TimeZone),
		QuietStartMinutes:      cloneIntPointer(dto.QuietStartMinutes),
		QuietEndMinutes:        cloneIntPointer(dto.QuietEndMinutes),
		NotificationsMaxPerDay: dto.NotificationsMaxPerDay,
		MinSleepMinutes:        dto.MinSleepMinutes,
		MinSteps:               dto.MinSteps,
		MinActiveEnergyKcal:    dto.MinActiveEnergyKcal,
		MorningCheckinMinute:   dto.MorningCheckinTimeMinutes,
		EveningCheckinMinute:   dto.EveningCheckinTimeMinutes,
		VitaminsTimeMinute:     dto.VitaminsTimeMinutes,
	}
}

func cloneIntPointer(v *int) *int {
	if v == nil {
		return nil
	}
	copied := *v
	return &copied
}

func cloneStringPointer(v *string) *string {
	if v == nil {
		return nil
	}
	copied := *v
	return &copied
}
