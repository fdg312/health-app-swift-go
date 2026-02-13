-- Таблица дневных метрик (агрегированные данные за день)
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

COMMENT ON TABLE daily_metrics IS 'Агрегированные метрики здоровья за день';
COMMENT ON COLUMN daily_metrics.profile_id IS 'ID профиля';
COMMENT ON COLUMN daily_metrics.date IS 'Дата (YYYY-MM-DD)';
COMMENT ON COLUMN daily_metrics.payload IS 'JSON с данными: sleep, activity, body, heart, nutrition, intakes';

-- Таблица часовых метрик
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

COMMENT ON TABLE hourly_metrics IS 'Метрики здоровья по часам (шаги и пульс)';
COMMENT ON COLUMN hourly_metrics.profile_id IS 'ID профиля';
COMMENT ON COLUMN hourly_metrics.hour IS 'Начало часа (UTC)';
COMMENT ON COLUMN hourly_metrics.steps IS 'Количество шагов за час';
COMMENT ON COLUMN hourly_metrics.hr_min IS 'Минимальный пульс за час';
COMMENT ON COLUMN hourly_metrics.hr_max IS 'Максимальный пульс за час';
COMMENT ON COLUMN hourly_metrics.hr_avg IS 'Средний пульс за час';

-- Таблица сегментов сна
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

COMMENT ON TABLE sleep_segments IS 'Сегменты сна с указанием стадии';
COMMENT ON COLUMN sleep_segments.profile_id IS 'ID профиля';
COMMENT ON COLUMN sleep_segments.start IS 'Начало сегмента сна';
COMMENT ON COLUMN sleep_segments."end" IS 'Конец сегмента сна';
COMMENT ON COLUMN sleep_segments.stage IS 'Стадия сна: rem, deep, core, awake';

-- Таблица тренировок
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

COMMENT ON TABLE workouts IS 'Сессии тренировок';
COMMENT ON COLUMN workouts.profile_id IS 'ID профиля';
COMMENT ON COLUMN workouts.start IS 'Начало тренировки';
COMMENT ON COLUMN workouts."end" IS 'Конец тренировки';
COMMENT ON COLUMN workouts.label IS 'Тип тренировки (strength, run, etc.)';
COMMENT ON COLUMN workouts.calories_kcal IS 'Количество сожжённых калорий';
