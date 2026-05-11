-- Migration 002: activity events

BEGIN;

CREATE TABLE IF NOT EXISTS activity_events (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    event_type  TEXT        NOT NULL,
    repo_id     UUID        REFERENCES repositories (id) ON DELETE CASCADE,
    payload     JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS activity_events_actor_id_idx  ON activity_events (actor_id);
CREATE INDEX IF NOT EXISTS activity_events_created_at_idx ON activity_events (created_at DESC);
CREATE INDEX IF NOT EXISTS activity_events_repo_id_idx   ON activity_events (repo_id);

COMMIT;
