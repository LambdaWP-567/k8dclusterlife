-- +goose Up

CREATE TABLE sessions (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    email       TEXT NOT NULL,
    provider    TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX sessions_expires ON sessions(expires_at);

CREATE TABLE clusters (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL UNIQUE,
    secret_name     TEXT NOT NULL,
    secret_namespace TEXT NOT NULL DEFAULT 'default',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at    TIMESTAMPTZ,
    reachable       BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE healing_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id      UUID NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
    problem_id      TEXT NOT NULL,
    problem_kind    TEXT NOT NULL,
    problem_name    TEXT NOT NULL,
    problem_ns      TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'running',
    autonomy_mode   TEXT NOT NULL DEFAULT 'MANUAL',
    result          TEXT,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at     TIMESTAMPTZ
);

CREATE INDEX healing_sessions_cluster ON healing_sessions(cluster_id);
CREATE INDEX healing_sessions_status  ON healing_sessions(status);

CREATE TABLE healing_actions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL REFERENCES healing_sessions(id) ON DELETE CASCADE,
    tool_name       TEXT NOT NULL,
    tool_input      JSONB NOT NULL,
    tool_output     JSONB,
    approved        BOOLEAN,
    executed_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX healing_actions_session ON healing_actions(session_id);

CREATE TABLE healing_snapshots (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL REFERENCES healing_sessions(id) ON DELETE CASCADE,
    resource_kind   TEXT NOT NULL,
    resource_name   TEXT NOT NULL,
    resource_ns     TEXT NOT NULL,
    snapshot        JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE test_results (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario        TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'running',
    playwright_report TEXT,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at     TIMESTAMPTZ
);

CREATE TABLE settings (
    key     TEXT PRIMARY KEY,
    value   TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO settings (key, value) VALUES
    ('autonomy_mode',       'MANUAL'),
    ('countdown_seconds',   '60'),
    ('refresh_interval_s',  '30'),
    ('teams_webhook_url',   ''),
    ('claude_model',        'claude-sonnet-4-6');

-- +goose Down

DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS test_results;
DROP TABLE IF EXISTS healing_snapshots;
DROP TABLE IF EXISTS healing_actions;
DROP TABLE IF EXISTS healing_sessions;
DROP TABLE IF EXISTS clusters;
DROP TABLE IF EXISTS sessions;
