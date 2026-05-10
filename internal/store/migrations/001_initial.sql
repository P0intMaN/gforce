-- Migration 001: initial schema

BEGIN;

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- updated_at trigger ---------------------------------------------------------

CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- users ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS users (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(39)  NOT NULL UNIQUE,
    email         TEXT         NOT NULL UNIQUE,
    password_hash TEXT         NOT NULL,
    display_name  TEXT,
    avatar_url    TEXT,
    bio           TEXT,
    is_admin      BOOLEAN      NOT NULL DEFAULT false,
    is_active     BOOLEAN      NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_username   ON users (username);
CREATE INDEX IF NOT EXISTS idx_users_email      ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users (created_at DESC);

DROP TRIGGER IF EXISTS set_users_updated_at ON users;
CREATE TRIGGER set_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- repositories ---------------------------------------------------------------

CREATE TABLE IF NOT EXISTS repositories (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id       UUID         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name           VARCHAR(100) NOT NULL,
    description    TEXT,
    is_private     BOOLEAN      NOT NULL DEFAULT false,
    default_branch VARCHAR(255) NOT NULL DEFAULT 'main',
    disk_path      TEXT         NOT NULL,
    fork_of        UUID         REFERENCES repositories (id) ON DELETE SET NULL,
    star_count     INT          NOT NULL DEFAULT 0,
    fork_count     INT          NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT repositories_owner_name_unique UNIQUE (owner_id, name)
);

CREATE INDEX IF NOT EXISTS idx_repositories_owner_id   ON repositories (owner_id);
CREATE INDEX IF NOT EXISTS idx_repositories_created_at ON repositories (created_at DESC);

DROP TRIGGER IF EXISTS set_repositories_updated_at ON repositories;
CREATE TRIGGER set_repositories_updated_at
    BEFORE UPDATE ON repositories
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- ssh_keys -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS ssh_keys (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title       TEXT        NOT NULL,
    public_key  TEXT        NOT NULL UNIQUE,
    fingerprint TEXT        NOT NULL UNIQUE,
    last_used_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ssh_keys_user_id     ON ssh_keys (user_id);
CREATE INDEX IF NOT EXISTS idx_ssh_keys_fingerprint ON ssh_keys (fingerprint);

COMMIT;
