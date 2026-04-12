-- +goose Up
CREATE TABLE download_clients (
    id              SERIAL PRIMARY KEY,
    name            TEXT NOT NULL,
    implementation  TEXT NOT NULL,
    settings        JSONB NOT NULL DEFAULT '{}',
    enable          BOOLEAN NOT NULL DEFAULT TRUE,
    priority        INTEGER NOT NULL DEFAULT 1,
    remove_completed_downloads BOOLEAN NOT NULL DEFAULT TRUE,
    remove_failed_downloads    BOOLEAN NOT NULL DEFAULT TRUE,
    added           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE download_clients;
