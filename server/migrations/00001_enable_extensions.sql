-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- +goose Down
-- no-op
SELECT 1;
