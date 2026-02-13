-- Intakes: Water & Supplements tracking

-- Supplements (витамины/добавки)
CREATE TABLE IF NOT EXISTS supplements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_supplements_profile_created
    ON supplements(profile_id, created_at DESC);

-- Supplement Components (компоненты добавок: витамин C, магний и т.д.)
CREATE TABLE IF NOT EXISTS supplement_components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    supplement_id UUID NOT NULL REFERENCES supplements(id) ON DELETE CASCADE,
    nutrient_key TEXT NOT NULL,        -- например "vitamin_c", "magnesium"
    hk_identifier TEXT,                -- например "dietaryVitaminC" (для iOS HealthKit)
    amount REAL NOT NULL,
    unit TEXT NOT NULL,                -- "mg", "mcg", "g", "ml", "IU"
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_supplement_components_supplement
    ON supplement_components(supplement_id);

-- Water Intakes (приём воды)
CREATE TABLE IF NOT EXISTS water_intakes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    taken_at TIMESTAMPTZ NOT NULL,
    amount_ml INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_water_intakes_profile_taken
    ON water_intakes(profile_id, taken_at DESC);

-- Supplement Intakes (отметки о приёме добавок)
CREATE TABLE IF NOT EXISTS supplement_intakes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    supplement_id UUID NOT NULL REFERENCES supplements(id) ON DELETE CASCADE,
    taken_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('taken', 'skipped')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Одна отметка в день на добавку
    UNIQUE(profile_id, supplement_id, DATE(taken_at AT TIME ZONE 'UTC'))
);

CREATE INDEX IF NOT EXISTS idx_supplement_intakes_profile_taken
    ON supplement_intakes(profile_id, taken_at DESC);
