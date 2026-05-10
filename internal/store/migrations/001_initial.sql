-- Migration 001: initial schema
-- Creates the core tables for users, repositories, and SSH keys.

BEGIN;

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- users -----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username      TEXT        NOT NULL,
    email         TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT users_username_unique UNIQUE (username),
    CONSTRAINT users_email_unique    UNIQUE (email),
    CONSTRAINT users_username_len    CHECK  (char_length(username) BETWEEN 1 AND 64),
    CONSTRAINT users_email_len       CHECK  (char_length(email)    <= 254)
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users (username);
CREATE INDEX IF NOT EXISTS idx_users_email    ON users (email);

-- repositories ----------------------------------------------------------------

CREATE TABLE IF NOT EXISTS repositories (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id       UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name           TEXT        NOT NULL,
    description    TEXT        NOT NULL DEFAULT '',
    is_private     BOOLEAN     NOT NULL DEFAULT FALSE,
    default_branch TEXT        NOT NULL DEFAULT 'main',
    disk_path      TEXT        NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT repositories_owner_name_unique UNIQUE (owner_id, name),
    CONSTRAINT repositories_name_len          CHECK  (char_length(name) BETWEEN 1 AND 128),
    CONSTRAINT repositories_disk_path_nonempty CHECK (char_length(disk_path) > 0)
);

CREATE INDEX IF NOT EXISTS idx_repositories_owner_id  ON repositories (owner_id);
CREATE INDEX IF NOT EXISTS idx_repositories_is_private ON repositories (is_private);
CREATE INDEX IF NOT EXISTS idx_repositories_created_at ON repositories (created_at DESC);

-- ssh_keys --------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS ssh_keys (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title       TEXT        NOT NULL,
    public_key  TEXT        NOT NULL,
    fingerprint TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT ssh_keys_fingerprint_unique UNIQUE (fingerprint),
    CONSTRAINT ssh_keys_title_len          CHECK  (char_length(title) BETWEEN 1 AND 255)
);

CREATE INDEX IF NOT EXISTS idx_ssh_keys_user_id     ON ssh_keys (user_id);
CREATE INDEX IF NOT EXISTS idx_ssh_keys_fingerprint ON ssh_keys (fingerprint);

COMMIT;
