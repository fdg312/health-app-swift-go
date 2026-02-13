-- +goose Up
CREATE TABLE IF NOT EXISTS daily_metrics (
    profile_id UUID NOT NULL,
    date DATE NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (profile_id, date),
    FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_daily_metrics_date ON daily_metrics(date);

CREATE TABLE IF NOT EXISTS hourly_metrics (
    profile_id UUID NOT NULL,
    hour TIMESTAMPTZ NOT NULL,
    steps INT NULL,
    hr_min INT NULL,
    hr_max INT NULL,
    hr_avg INT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (profile_id, hour),
    FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE,
    CHECK (hr_min IS NULL OR hr_min > 0),
    CHECK (hr_max IS NULL OR hr_max > 0),
    CHECK (hr_avg IS NULL OR hr_avg > 0),
    CHECK (steps IS NULL OR steps >= 0)
);

CREATE INDEX IF NOT EXISTS idx_hourly_metrics_hour ON hourly_metrics(hour);

CREATE TABLE IF NOT EXISTS sleep_segments (
    profile_id UUID NOT NULL,
    start TIMESTAMPTZ NOT NULL,
    "end" TIMESTAMPTZ NOT NULL,
    stage TEXT NOT NULL CHECK (stage IN ('rem', 'deep', 'core', 'awake')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (profile_id, start, "end", stage),
    FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE,
    CHECK (start < "end")
);

CREATE INDEX IF NOT EXISTS idx_sleep_segments_profile_start ON sleep_segments(profile_id, start);

CREATE TABLE IF NOT EXISTS workouts (
    profile_id UUID NOT NULL,
    start TIMESTAMPTZ NOT NULL,
    "end" TIMESTAMPTZ NOT NULL,
    label TEXT NOT NULL,
    calories_kcal INT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (profile_id, start, "end", label),
    FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE,
    CHECK (start < "end"),
    CHECK (calories_kcal IS NULL OR calories_kcal >= 0)
);

CREATE INDEX IF NOT EXISTS idx_workouts_profile_start ON workouts(profile_id, start);

-- +goose Down
DROP TABLE IF EXISTS workouts;
DROP TABLE IF EXISTS sleep_segments;
DROP TABLE IF EXISTS hourly_metrics;
DROP TABLE IF EXISTS daily_metrics;
