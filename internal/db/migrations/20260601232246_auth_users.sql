-- +goose Up
-- +goose StatementBegin

CREATE SCHEMA IF NOT EXISTS auth_schema;

SET search_path = auth_schema;

CREATE TABLE IF NOT EXISTS users (
    id            UUID        NOT NULL DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    is_verified   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT users_pkey        PRIMARY KEY (id),
    CONSTRAINT users_email_uq    UNIQUE      (email),
    CONSTRAINT users_email_check CHECK       (char_length(email) <= 254)
);

CREATE INDEX IF NOT EXISTS idx_users_email
    ON users (email);

RESET search_path;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS auth_schema.users CASCADE;

-- +goose StatementEnd