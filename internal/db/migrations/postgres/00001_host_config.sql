-- +goose Up
CREATE TABLE host_config (
    id              SMALLINT PRIMARY KEY CHECK (id = 1),
    api_key         TEXT NOT NULL,
    auth_mode       TEXT NOT NULL DEFAULT 'forms',
    migration_state TEXT NOT NULL DEFAULT 'clean',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE host_config;
