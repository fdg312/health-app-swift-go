-- Checkins table
-- Stores morning and evening check-ins with scores, tags, and notes

CREATE TABLE IF NOT EXISTS checkins (
    id UUID PRIMARY KEY,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('morning', 'evening')),
    score INT NOT NULL CHECK (score >= 1 AND score <= 5),
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (profile_id, date, type)
);

-- Index for querying by profile and date range
CREATE INDEX IF NOT EXISTS idx_checkins_profile_date ON checkins(profile_id, date DESC);
