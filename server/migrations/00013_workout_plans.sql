-- +goose Up
-- +goose StatementBegin
CREATE TABLE workout_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id TEXT NOT NULL,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    goal TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_workout_plans_owner_profile ON workout_plans(owner_user_id, profile_id);
CREATE UNIQUE INDEX idx_workout_plans_active_unique ON workout_plans(owner_user_id, profile_id) WHERE is_active = true;

CREATE TABLE workout_plan_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL REFERENCES workout_plans(id) ON DELETE CASCADE,
    owner_user_id TEXT NOT NULL,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK(kind IN ('run','walk','strength','morning','core','other')),
    time_minutes INT NOT NULL CHECK(time_minutes BETWEEN 0 AND 1439),
    days_mask INT NOT NULL DEFAULT 127 CHECK(days_mask BETWEEN 0 AND 127),
    duration_min INT NOT NULL CHECK(duration_min BETWEEN 5 AND 240),
    intensity TEXT NOT NULL DEFAULT 'medium' CHECK(intensity IN ('low','medium','high')),
    note TEXT NOT NULL DEFAULT '',
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_workout_plan_items_owner_profile ON workout_plan_items(owner_user_id, profile_id);
CREATE INDEX idx_workout_plan_items_plan ON workout_plan_items(plan_id);
CREATE UNIQUE INDEX idx_workout_plan_items_unique ON workout_plan_items(owner_user_id, profile_id, kind, time_minutes, days_mask);

CREATE TABLE workout_completions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id TEXT NOT NULL,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    plan_item_id UUID NOT NULL REFERENCES workout_plan_items(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK(status IN ('done','skipped')),
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_workout_completions_unique ON workout_completions(owner_user_id, profile_id, date, plan_item_id);
CREATE INDEX idx_workout_completions_date ON workout_completions(owner_user_id, profile_id, date);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS workout_completions;
DROP TABLE IF EXISTS workout_plan_items;
DROP TABLE IF EXISTS workout_plans;
-- +goose StatementEnd
