-- Sources table: links, notes, images attached to profiles/checkins
-- Run this after profiles.sql and checkins.sql

CREATE TABLE IF NOT EXISTS sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('link', 'note', 'image')),
    title TEXT,
    text TEXT,
    url TEXT,
    checkin_id UUID REFERENCES checkins(id) ON DELETE CASCADE,
    object_key TEXT,           -- S3 object key for images
    content_type TEXT,         -- MIME type (e.g., image/jpeg)
    size_bytes BIGINT,         -- File size in bytes
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index for listing sources by profile (most recent first)
CREATE INDEX IF NOT EXISTS idx_sources_profile_created
    ON sources(profile_id, created_at DESC);

-- Index for filtering sources by checkin
CREATE INDEX IF NOT EXISTS idx_sources_checkin
    ON sources(checkin_id)
    WHERE checkin_id IS NOT NULL;

-- Comments
COMMENT ON TABLE sources IS 'User-generated content: links, notes, and images attached to profiles or checkins';
COMMENT ON COLUMN sources.kind IS 'Type of source: link, note, or image';
COMMENT ON COLUMN sources.object_key IS 'S3 object key (images only)';
COMMENT ON COLUMN sources.content_type IS 'MIME type for images';
COMMENT ON COLUMN sources.checkin_id IS 'Optional reference to a checkin (for context/attachments)';
