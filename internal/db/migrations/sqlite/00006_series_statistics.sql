-- +goose Up
CREATE TABLE series_statistics (
    series_id                 INTEGER PRIMARY KEY REFERENCES series(id) ON DELETE CASCADE,
    episode_count             INTEGER NOT NULL DEFAULT 0,
    episode_file_count        INTEGER NOT NULL DEFAULT 0,
    monitored_episode_count   INTEGER NOT NULL DEFAULT 0,
    size_on_disk              INTEGER NOT NULL DEFAULT 0,
    updated_at                TEXT NOT NULL DEFAULT (datetime('now'))
);

-- +goose Down
DROP TABLE series_statistics;
