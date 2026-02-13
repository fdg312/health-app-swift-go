package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresSettingsStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresSettingsStorage(pool *pgxpool.Pool) *PostgresSettingsStorage {
	return &PostgresSettingsStorage{pool: pool}
}

func (s *PostgresSettingsStorage) GetSettings(ctx context.Context, ownerUserID string) (storage.Settings, bool, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)

	const query = `
		SELECT owner_user_id, time_zone, quiet_start_minutes, quiet_end_minutes,
		       notifications_max_per_day, min_sleep_minutes, min_steps, min_active_energy_kcal,
		       morning_checkin_time_minutes, evening_checkin_time_minutes, vitamins_time_minutes,
		       created_at, updated_at
		FROM user_settings
		WHERE owner_user_id = $1
	`

	var row storage.Settings
	err := s.pool.QueryRow(ctx, query, ownerUserID).Scan(
		&row.OwnerUserID,
		&row.TimeZone,
		&row.QuietStartMinutes,
		&row.QuietEndMinutes,
		&row.NotificationsMaxPerDay,
		&row.MinSleepMinutes,
		&row.MinSteps,
		&row.MinActiveEnergyKcal,
		&row.MorningCheckinMinute,
		&row.EveningCheckinMinute,
		&row.VitaminsTimeMinute,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return storage.Settings{}, false, nil
		}
		return storage.Settings{}, false, err
	}

	return row, true, nil
}

func (s *PostgresSettingsStorage) UpsertSettings(ctx context.Context, ownerUserID string, in storage.Settings) (storage.Settings, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)

	const query = `
		INSERT INTO user_settings (
			owner_user_id, time_zone, quiet_start_minutes, quiet_end_minutes,
			notifications_max_per_day, min_sleep_minutes, min_steps, min_active_energy_kcal,
			morning_checkin_time_minutes, evening_checkin_time_minutes, vitamins_time_minutes,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		ON CONFLICT (owner_user_id) DO UPDATE SET
			time_zone = EXCLUDED.time_zone,
			quiet_start_minutes = EXCLUDED.quiet_start_minutes,
			quiet_end_minutes = EXCLUDED.quiet_end_minutes,
			notifications_max_per_day = EXCLUDED.notifications_max_per_day,
			min_sleep_minutes = EXCLUDED.min_sleep_minutes,
			min_steps = EXCLUDED.min_steps,
			min_active_energy_kcal = EXCLUDED.min_active_energy_kcal,
			morning_checkin_time_minutes = EXCLUDED.morning_checkin_time_minutes,
			evening_checkin_time_minutes = EXCLUDED.evening_checkin_time_minutes,
			vitamins_time_minutes = EXCLUDED.vitamins_time_minutes,
			updated_at = NOW()
		RETURNING owner_user_id, time_zone, quiet_start_minutes, quiet_end_minutes,
		          notifications_max_per_day, min_sleep_minutes, min_steps, min_active_energy_kcal,
		          morning_checkin_time_minutes, evening_checkin_time_minutes, vitamins_time_minutes,
		          created_at, updated_at
	`

	var out storage.Settings
	err := s.pool.QueryRow(ctx, query,
		ownerUserID,
		in.TimeZone,
		in.QuietStartMinutes,
		in.QuietEndMinutes,
		in.NotificationsMaxPerDay,
		in.MinSleepMinutes,
		in.MinSteps,
		in.MinActiveEnergyKcal,
		in.MorningCheckinMinute,
		in.EveningCheckinMinute,
		in.VitaminsTimeMinute,
	).Scan(
		&out.OwnerUserID,
		&out.TimeZone,
		&out.QuietStartMinutes,
		&out.QuietEndMinutes,
		&out.NotificationsMaxPerDay,
		&out.MinSleepMinutes,
		&out.MinSteps,
		&out.MinActiveEnergyKcal,
		&out.MorningCheckinMinute,
		&out.EveningCheckinMinute,
		&out.VitaminsTimeMinute,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return storage.Settings{}, err
	}

	return out, nil
}
