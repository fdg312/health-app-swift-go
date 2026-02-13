-- +goose Up
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    source_date DATE,
    severity TEXT NOT NULL CHECK (severity IN ('info', 'warn')) DEFAULT 'info',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    read_at TIMESTAMPTZ,
    UNIQUE(profile_id, kind, source_date)
);

CREATE INDEX IF NOT EXISTS idx_notifications_profile_created
    ON notifications(profile_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_profile_read
    ON notifications(profile_id, read_at);

-- +goose Down
DROP TABLE IF EXISTS notifications;
