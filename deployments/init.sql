--  Executed automatically by postgres on first container start.
--  Creates one schema per service to match the database ownership model.

CREATE SCHEMA IF NOT EXISTS auth_schema;
CREATE SCHEMA IF NOT EXISTS user_schema;
CREATE SCHEMA IF NOT EXISTS chat_schema;
CREATE SCHEMA IF NOT EXISTS media_schema;

--   Auth Schema 
CREATE TABLE auth_schema.users (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email          VARCHAR(255) UNIQUE NOT NULL,
    password_hash  VARCHAR(255) NOT NULL,
    email_verified BOOLEAN      DEFAULT FALSE,
    created_at     TIMESTAMPTZ  DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  DEFAULT NOW()
);

CREATE TABLE auth_schema.refresh_tokens (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL REFERENCES auth_schema.users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ  NOT NULL,
    revoked    BOOLEAN      DEFAULT FALSE,
    created_at TIMESTAMPTZ  DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_id    ON auth_schema.refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON auth_schema.refresh_tokens(token_hash);

--   User Schema 
CREATE TABLE user_schema.profiles (
    user_id      UUID         PRIMARY KEY REFERENCES auth_schema.users(id) ON DELETE CASCADE,
    username     VARCHAR(50)  UNIQUE NOT NULL,
    display_name VARCHAR(100),
    avatar_url   TEXT,
    bio          TEXT,
    last_seen    TIMESTAMPTZ,
    is_online    BOOLEAN      DEFAULT FALSE,
    created_at   TIMESTAMPTZ  DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  DEFAULT NOW()
);

CREATE INDEX idx_profiles_username ON user_schema.profiles(username);

--   Chat Schema 
CREATE TABLE chat_schema.conversations (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    type       VARCHAR(20) NOT NULL DEFAULT 'direct',
    name       VARCHAR(100),
    created_by UUID        REFERENCES auth_schema.users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE chat_schema.conversation_members (
    conversation_id UUID        NOT NULL REFERENCES chat_schema.conversations(id) ON DELETE CASCADE,
    user_id         UUID        NOT NULL REFERENCES auth_schema.users(id) ON DELETE CASCADE,
    role            VARCHAR(20) DEFAULT 'member',
    joined_at       TIMESTAMPTZ DEFAULT NOW(),
    last_read_at    TIMESTAMPTZ,
    PRIMARY KEY (conversation_id, user_id)
);

CREATE TABLE chat_schema.messages (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID        NOT NULL REFERENCES chat_schema.conversations(id) ON DELETE CASCADE,
    sender_id       UUID        NOT NULL REFERENCES auth_schema.users(id),
    content         TEXT,
    type            VARCHAR(20) DEFAULT 'text',
    reply_to        UUID        REFERENCES chat_schema.messages(id),
    edited          BOOLEAN     DEFAULT FALSE,
    deleted         BOOLEAN     DEFAULT FALSE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_messages_conversation ON chat_schema.messages(conversation_id, created_at DESC);
CREATE INDEX idx_messages_sender       ON chat_schema.messages(sender_id);

CREATE TABLE chat_schema.message_reactions (
    message_id UUID        NOT NULL REFERENCES chat_schema.messages(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES auth_schema.users(id) ON DELETE CASCADE,
    emoji      VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id, emoji)
);

--   Media Schema 
CREATE TABLE media_schema.uploads (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID         NOT NULL REFERENCES auth_schema.users(id),
    message_id   UUID         REFERENCES chat_schema.messages(id),
    filename     VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size_bytes   BIGINT       NOT NULL,
    storage_key  TEXT         NOT NULL,
    created_at   TIMESTAMPTZ  DEFAULT NOW()
);

CREATE INDEX idx_uploads_user    ON media_schema.uploads(user_id);
CREATE INDEX idx_uploads_message ON media_schema.uploads(message_id);