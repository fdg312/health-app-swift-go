-- +goose Up
CREATE TABLE IF NOT EXISTS email_otps (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL,
    code_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 5,
    last_sent_at TIMESTAMPTZ NOT NULL,
    send_count INT NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_email_otps_email_created
    ON email_otps(email, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_email_otps_email_expires
    ON email_otps(email, expires_at);

-- +goose Down
DROP TABLE IF EXISTS email_otps;
