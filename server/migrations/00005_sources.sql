-- +goose Up
CREATE TABLE IF NOT EXISTS sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('link', 'note', 'image')),
    title TEXT,
    text TEXT,
    url TEXT,
    checkin_id UUID REFERENCES checkins(id) ON DELETE CASCADE,
    object_key TEXT,
    content_type TEXT,
    size_bytes BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sources_profile_created ON sources(profile_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sources_checkin ON sources(checkin_id) WHERE checkin_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS sources;
