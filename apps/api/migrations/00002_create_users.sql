-- +goose Up
-- Emails are stored normalized (lowercase, trimmed) by the
-- application (domain.NormalizeEmail), so the unique index does not
-- need a lower() expression.

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL CHECK (char_length(email) <= 320),
    password_hash TEXT NOT NULL CHECK (char_length(password_hash) <= 256),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_email_key ON users (email);
