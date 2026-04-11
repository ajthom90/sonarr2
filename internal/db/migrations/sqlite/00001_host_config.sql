-- +goose Up
CREATE TABLE host_config (
    id              INTEGER PRIMARY KEY CHECK (id = 1),
    api_key         TEXT NOT NULL,
    auth_mode       TEXT NOT NULL DEFAULT 'forms',
    migration_state TEXT NOT NULL DEFAULT 'clean',
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

-- +goose Down
DROP TABLE host_config;
