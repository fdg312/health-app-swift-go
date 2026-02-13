-- +goose Up
-- +goose StatementBegin
CREATE TABLE nutrition_targets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id TEXT NOT NULL,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    calories_kcal INT NOT NULL CHECK (calories_kcal >= 800 AND calories_kcal <= 6000),
    protein_g INT NOT NULL CHECK (protein_g >= 0 AND protein_g <= 400),
    fat_g INT NOT NULL CHECK (fat_g >= 0 AND fat_g <= 400),
    carbs_g INT NOT NULL CHECK (carbs_g >= 0 AND carbs_g <= 400),
    calcium_mg INT NOT NULL DEFAULT 0 CHECK (calcium_mg >= 0 AND calcium_mg <= 5000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX nutrition_targets_owner_profile_idx ON nutrition_targets(owner_user_id, profile_id);
CREATE INDEX nutrition_targets_lookup_idx ON nutrition_targets(owner_user_id, profile_id);

COMMENT ON TABLE nutrition_targets IS 'Nutrition targets/goals per profile';
COMMENT ON COLUMN nutrition_targets.calories_kcal IS 'Daily calorie target (800-6000 kcal)';
COMMENT ON COLUMN nutrition_targets.protein_g IS 'Daily protein target (0-400g)';
COMMENT ON COLUMN nutrition_targets.fat_g IS 'Daily fat target (0-400g)';
COMMENT ON COLUMN nutrition_targets.carbs_g IS 'Daily carbs target (0-400g)';
COMMENT ON COLUMN nutrition_targets.calcium_mg IS 'Daily calcium target (0-5000mg)';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS nutrition_targets;
-- +goose StatementEnd
