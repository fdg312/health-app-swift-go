-- Reports table (PDF/CSV export metadata)
CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    format TEXT NOT NULL CHECK (format IN ('pdf', 'csv')),
    from_date DATE NOT NULL,
    to_date DATE NOT NULL,
    object_key TEXT NULL, -- S3 object key (NULL if using local/memory storage)
    size_bytes BIGINT NOT NULL DEFAULT 0,
    status TEXT NOT NULL CHECK (status IN ('ready', 'failed')) DEFAULT 'ready',
    error TEXT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_reports_profile_created ON reports(profile_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reports_profile_dates ON reports(profile_id, from_date, to_date);

-- Comments
COMMENT ON TABLE reports IS 'Reports metadata (PDF/CSV exports)';
COMMENT ON COLUMN reports.object_key IS 'S3 object key if using object storage, NULL for local/memory mode';
COMMENT ON COLUMN reports.status IS 'Report generation status: ready or failed';
