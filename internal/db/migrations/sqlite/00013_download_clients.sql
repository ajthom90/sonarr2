-- +goose Up
CREATE TABLE download_clients (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL,
    implementation  TEXT NOT NULL,
    settings        TEXT NOT NULL DEFAULT '{}',
    enable          INTEGER NOT NULL DEFAULT 1,
    priority        INTEGER NOT NULL DEFAULT 1,
    remove_completed_downloads INTEGER NOT NULL DEFAULT 1,
    remove_failed_downloads    INTEGER NOT NULL DEFAULT 1,
    added           TEXT NOT NULL DEFAULT (datetime('now'))
);

-- +goose Down
DROP TABLE download_clients;
