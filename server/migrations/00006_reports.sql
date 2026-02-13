-- +goose Up
CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    format TEXT NOT NULL CHECK (format IN ('pdf', 'csv')),
    from_date DATE NOT NULL,
    to_date DATE NOT NULL,
    object_key TEXT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    status TEXT NOT NULL CHECK (status IN ('ready', 'failed')) DEFAULT 'ready',
    error TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reports_profile_created ON reports(profile_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reports_profile_dates ON reports(profile_id, from_date, to_date);

-- +goose Down
DROP TABLE IF EXISTS reports;
