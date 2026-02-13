-- Таблица профилей
CREATE TABLE IF NOT EXISTS profiles (
    id UUID PRIMARY KEY,
    owner_user_id TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('owner', 'guest')),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ограничение: только один owner профиль на owner_user_id
    UNIQUE (owner_user_id, type) WHERE type = 'owner'
);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_profiles_owner_user_id ON profiles(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_profiles_type ON profiles(type);

-- Комментарии
COMMENT ON TABLE profiles IS 'Профили пользователей (owner и guest)';
COMMENT ON COLUMN profiles.id IS 'Уникальный идентификатор профиля';
COMMENT ON COLUMN profiles.owner_user_id IS 'ID владельца (default для MVP)';
COMMENT ON COLUMN profiles.type IS 'Тип профиля: owner или guest';
COMMENT ON COLUMN profiles.name IS 'Имя профиля';
