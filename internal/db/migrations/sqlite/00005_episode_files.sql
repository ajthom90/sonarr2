-- +goose Up
CREATE TABLE episode_files (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    series_id      INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    season_number  INTEGER NOT NULL,
    relative_path  TEXT NOT NULL,
    size           INTEGER NOT NULL,
    date_added     TEXT NOT NULL DEFAULT (datetime('now')),
    release_group  TEXT NOT NULL DEFAULT '',
    quality_name   TEXT NOT NULL DEFAULT '',
    created_at     TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at     TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX episode_files_series_season_idx ON episode_files (series_id, season_number);

-- +goose Down
DROP TABLE episode_files;
