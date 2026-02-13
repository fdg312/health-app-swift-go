-- +goose Up
CREATE TABLE IF NOT EXISTS supplements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_supplements_profile_created
    ON supplements(profile_id, created_at DESC);

CREATE TABLE IF NOT EXISTS supplement_components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    supplement_id UUID NOT NULL REFERENCES supplements(id) ON DELETE CASCADE,
    nutrient_key TEXT NOT NULL,
    hk_identifier TEXT,
    amount REAL NOT NULL,
    unit TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_supplement_components_supplement
    ON supplement_components(supplement_id);

CREATE TABLE IF NOT EXISTS water_intakes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    taken_at TIMESTAMPTZ NOT NULL,
    amount_ml INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_water_intakes_profile_taken
    ON water_intakes(profile_id, taken_at DESC);

CREATE TABLE IF NOT EXISTS supplement_intakes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    supplement_id UUID NOT NULL REFERENCES supplements(id) ON DELETE CASCADE,
    taken_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('taken', 'skipped')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_supplement_intakes_profile_taken
    ON supplement_intakes(profile_id, taken_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_supplement_intakes_unique_daily
    ON supplement_intakes(profile_id, supplement_id, (DATE(taken_at AT TIME ZONE 'UTC')));

-- +goose Down
DROP TABLE IF EXISTS supplement_intakes;
DROP TABLE IF EXISTS water_intakes;
DROP TABLE IF EXISTS supplement_components;
DROP TABLE IF EXISTS supplements;
