-- Notifications/Inbox table
-- Stores server-side notifications for users

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,  -- low_sleep, low_activity, missing_morning_checkin, missing_evening_checkin
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    source_date DATE,  -- date this notification relates to (nullable)
    severity TEXT NOT NULL CHECK (severity IN ('info', 'warn')) DEFAULT 'info',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at TIMESTAMPTZ,
    
    -- Prevent duplicate notifications for same profile/kind/date
    UNIQUE(profile_id, kind, source_date)
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_notifications_profile_created 
    ON notifications(profile_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_profile_read 
    ON notifications(profile_id, read_at);
