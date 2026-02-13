-- +goose Up
CREATE TABLE IF NOT EXISTS chat_messages (
    id UUID PRIMARY KEY,
    owner_user_id TEXT NOT NULL,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_owner_profile_created
    ON chat_messages(owner_user_id, profile_id, created_at DESC);

CREATE TABLE IF NOT EXISTS ai_proposals (
    id UUID PRIMARY KEY,
    owner_user_id TEXT NOT NULL,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'applied', 'rejected')),
    kind TEXT NOT NULL CHECK (kind IN ('settings_update', 'vitamins_schedule', 'workout_plan', 'nutrition_plan', 'generic')),
    title TEXT NOT NULL,
    summary TEXT NOT NULL,
    payload JSONB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ai_proposals_owner_profile_created
    ON ai_proposals(owner_user_id, profile_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS ai_proposals;
DROP TABLE IF EXISTS chat_messages;
