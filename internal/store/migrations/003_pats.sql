-- Migration 003: personal access tokens

BEGIN;

CREATE TABLE IF NOT EXISTS personal_access_tokens (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name         TEXT        NOT NULL,
    token_hash   TEXT        NOT NULL UNIQUE,
    prefix       TEXT        NOT NULL,
    scopes       TEXT[]      NOT NULL DEFAULT '{repo:read,repo:write}',
    last_used_at TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS pats_user_id_idx ON personal_access_tokens (user_id);
CREATE INDEX IF NOT EXISTS pats_prefix_idx  ON personal_access_tokens (prefix);

COMMIT;
