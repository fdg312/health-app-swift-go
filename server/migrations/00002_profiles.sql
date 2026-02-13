-- +goose Up
CREATE TABLE IF NOT EXISTS profiles (
    id UUID PRIMARY KEY,
    owner_user_id TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('owner', 'guest')),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_profiles_owner_user_id ON profiles(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_profiles_type ON profiles(type);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_profiles_owner_by_user_and_type
    ON profiles(owner_user_id, type)
    WHERE type = 'owner';

COMMENT ON TABLE profiles IS 'User profiles (owner + guest)';
COMMENT ON COLUMN profiles.owner_user_id IS 'Owner user id from JWT sub';
COMMENT ON COLUMN profiles.type IS 'Profile type: owner or guest';

-- +goose Down
DROP TABLE IF EXISTS profiles;
