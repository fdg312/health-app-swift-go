-- +goose Up
-- +goose StatementBegin

-- Food preferences: user-defined foods with macros
CREATE TABLE food_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id TEXT NOT NULL,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (char_length(name) >= 1 AND char_length(name) <= 80),
    tags TEXT[] NOT NULL DEFAULT '{}',
    kcal_per_100g INT NOT NULL DEFAULT 0 CHECK (kcal_per_100g >= 0 AND kcal_per_100g <= 1000),
    protein_g_per_100g INT NOT NULL DEFAULT 0 CHECK (protein_g_per_100g >= 0 AND protein_g_per_100g <= 1000),
    fat_g_per_100g INT NOT NULL DEFAULT 0 CHECK (fat_g_per_100g >= 0 AND fat_g_per_100g <= 1000),
    carbs_g_per_100g INT NOT NULL DEFAULT 0 CHECK (carbs_g_per_100g >= 0 AND carbs_g_per_100g <= 1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX food_preferences_owner_profile_idx ON food_preferences(owner_user_id, profile_id);
CREATE UNIQUE INDEX food_preferences_name_unique_idx ON food_preferences(owner_user_id, profile_id, lower(name));

COMMENT ON TABLE food_preferences IS 'User-defined food items with nutritional information';
COMMENT ON COLUMN food_preferences.name IS 'Food name (1-80 chars)';
COMMENT ON COLUMN food_preferences.tags IS 'Tags like завтрак, перекус, веган, etc';
COMMENT ON COLUMN food_preferences.kcal_per_100g IS 'Calories per 100g (0-1000)';
COMMENT ON COLUMN food_preferences.protein_g_per_100g IS 'Protein grams per 100g (0-1000)';
COMMENT ON COLUMN food_preferences.fat_g_per_100g IS 'Fat grams per 100g (0-1000)';
COMMENT ON COLUMN food_preferences.carbs_g_per_100g IS 'Carbs grams per 100g (0-1000)';

-- Meal plans: weekly meal plan per profile
CREATE TABLE meal_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id TEXT NOT NULL,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    title TEXT NOT NULL CHECK (char_length(title) >= 1 AND char_length(title) <= 200),
    is_active BOOL NOT NULL DEFAULT true,
    from_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX meal_plans_active_unique_idx ON meal_plans(owner_user_id, profile_id) WHERE is_active = true;
CREATE INDEX meal_plans_owner_profile_idx ON meal_plans(owner_user_id, profile_id);

COMMENT ON TABLE meal_plans IS 'Meal plans (one active per profile)';
COMMENT ON COLUMN meal_plans.title IS 'Plan title/name';
COMMENT ON COLUMN meal_plans.is_active IS 'Only one active plan per profile allowed';
COMMENT ON COLUMN meal_plans.from_date IS 'Start date for plan (NULL = template)';

-- Meal plan items: meals for each day/slot
CREATE TABLE meal_plan_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id TEXT NOT NULL,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES meal_plans(id) ON DELETE CASCADE,
    day_index INT NOT NULL CHECK (day_index >= 0 AND day_index <= 6),
    meal_slot TEXT NOT NULL CHECK (meal_slot IN ('breakfast', 'lunch', 'dinner', 'snack')),
    title TEXT NOT NULL CHECK (char_length(title) >= 1 AND char_length(title) <= 200),
    notes TEXT NOT NULL DEFAULT '',
    approx_kcal INT NOT NULL DEFAULT 0 CHECK (approx_kcal >= 0 AND approx_kcal <= 10000),
    approx_protein_g INT NOT NULL DEFAULT 0 CHECK (approx_protein_g >= 0 AND approx_protein_g <= 1000),
    approx_fat_g INT NOT NULL DEFAULT 0 CHECK (approx_fat_g >= 0 AND approx_fat_g <= 1000),
    approx_carbs_g INT NOT NULL DEFAULT 0 CHECK (approx_carbs_g >= 0 AND approx_carbs_g <= 1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX meal_plan_items_plan_idx ON meal_plan_items(owner_user_id, profile_id, plan_id);
CREATE UNIQUE INDEX meal_plan_items_slot_unique_idx ON meal_plan_items(owner_user_id, profile_id, plan_id, day_index, meal_slot);

COMMENT ON TABLE meal_plan_items IS 'Individual meals in a meal plan (7 days x 4 slots max)';
COMMENT ON COLUMN meal_plan_items.day_index IS 'Day of week (0=Monday, 6=Sunday)';
COMMENT ON COLUMN meal_plan_items.meal_slot IS 'Meal type: breakfast, lunch, dinner, snack';
COMMENT ON COLUMN meal_plan_items.title IS 'Meal description (e.g. "Овсянка + банан")';
COMMENT ON COLUMN meal_plan_items.notes IS 'Additional notes/instructions';
COMMENT ON COLUMN meal_plan_items.approx_kcal IS 'Approximate calories for this meal';
COMMENT ON COLUMN meal_plan_items.approx_protein_g IS 'Approximate protein in grams';
COMMENT ON COLUMN meal_plan_items.approx_fat_g IS 'Approximate fat in grams';
COMMENT ON COLUMN meal_plan_items.approx_carbs_g IS 'Approximate carbs in grams';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS meal_plan_items;
DROP TABLE IF EXISTS meal_plans;
DROP TABLE IF EXISTS food_preferences;
-- +goose StatementEnd
