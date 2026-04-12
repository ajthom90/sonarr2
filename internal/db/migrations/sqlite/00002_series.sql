-- +goose Up
CREATE TABLE series (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    tvdb_id          INTEGER NOT NULL UNIQUE,
    title            TEXT NOT NULL,
    slug             TEXT NOT NULL UNIQUE,
    status           TEXT NOT NULL DEFAULT 'continuing',
    series_type      TEXT NOT NULL DEFAULT 'standard',
    path             TEXT NOT NULL UNIQUE,
    monitored        INTEGER NOT NULL DEFAULT 1,
    added            TEXT NOT NULL DEFAULT (datetime('now')),
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at       TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX series_title_idx ON series (title);

-- +goose Down
DROP TABLE series;
