-- +goose Up
CREATE TABLE notifications (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL,
    implementation  TEXT NOT NULL,
    settings        TEXT NOT NULL DEFAULT '{}',
    on_grab         INTEGER NOT NULL DEFAULT 1,
    on_download     INTEGER NOT NULL DEFAULT 1,
    on_health_issue INTEGER NOT NULL DEFAULT 1,
    tags            TEXT NOT NULL DEFAULT '[]',
    added           TEXT NOT NULL DEFAULT (datetime('now'))
);

-- +goose Down
DROP TABLE notifications;
