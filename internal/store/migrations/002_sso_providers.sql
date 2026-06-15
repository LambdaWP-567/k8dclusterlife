-- +goose Up

CREATE TABLE sso_providers (
    provider        TEXT PRIMARY KEY,
    enabled         BOOLEAN NOT NULL DEFAULT false,
    client_id       TEXT NOT NULL DEFAULT '',
    client_secret   TEXT NOT NULL DEFAULT '',
    tenant_id       TEXT NOT NULL DEFAULT '',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO sso_providers (provider) VALUES ('entra'), ('github'), ('google');

-- +goose Down

DROP TABLE IF EXISTS sso_providers;
