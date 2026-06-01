-- +goose Up
-- +goose StatementBegin

SET search_path = auth_schema;

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          UUID        NOT NULL DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL,
    token_hash  TEXT        NOT NULL,
    family      UUID        NOT NULL,
    used        BOOLEAN     NOT NULL DEFAULT FALSE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT refresh_tokens_pkey     PRIMARY KEY (id),
    CONSTRAINT refresh_tokens_hash_uq  UNIQUE      (token_hash),
    CONSTRAINT refresh_tokens_user_fk  FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rt_token_hash
    ON refresh_tokens (token_hash);

CREATE INDEX IF NOT EXISTS idx_rt_family
    ON refresh_tokens (family);

CREATE INDEX IF NOT EXISTS idx_rt_expires_at
    ON refresh_tokens (expires_at)
    WHERE used = FALSE;

CREATE INDEX IF NOT EXISTS idx_rt_user_active
    ON refresh_tokens (user_id, expires_at DESC)
    WHERE used = FALSE;

RESET search_path;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS auth_schema.refresh_tokens CASCADE;

-- +goose StatementEnd