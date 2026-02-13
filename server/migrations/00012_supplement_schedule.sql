-- +goose Up
CREATE TABLE IF NOT EXISTS supplement_schedules (
    id UUID PRIMARY KEY,
    owner_user_id TEXT NOT NULL,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    supplement_id UUID NOT NULL REFERENCES supplements(id) ON DELETE CASCADE,
    time_minutes INT NOT NULL,
    days_mask INT NOT NULL DEFAULT 127,
    is_enabled BOOL NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (time_minutes BETWEEN 0 AND 1439),
    CHECK (days_mask BETWEEN 0 AND 127)
);

CREATE INDEX IF NOT EXISTS idx_supplement_schedules_owner_profile
    ON supplement_schedules(owner_user_id, profile_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_supplement_schedules_unique
    ON supplement_schedules(owner_user_id, profile_id, supplement_id, time_minutes);

-- +goose Down
DROP TABLE IF EXISTS supplement_schedules;
