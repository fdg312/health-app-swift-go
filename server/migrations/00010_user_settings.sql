-- +goose Up
CREATE TABLE IF NOT EXISTS user_settings (
    owner_user_id TEXT PRIMARY KEY,
    time_zone TEXT NULL,
    quiet_start_minutes INT NULL,
    quiet_end_minutes INT NULL,
    notifications_max_per_day INT NOT NULL DEFAULT 4,
    min_sleep_minutes INT NOT NULL DEFAULT 420,
    min_steps INT NOT NULL DEFAULT 6000,
    min_active_energy_kcal INT NOT NULL DEFAULT 250,
    morning_checkin_time_minutes INT NOT NULL DEFAULT 540,
    evening_checkin_time_minutes INT NOT NULL DEFAULT 1260,
    vitamins_time_minutes INT NOT NULL DEFAULT 720,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS user_settings;
