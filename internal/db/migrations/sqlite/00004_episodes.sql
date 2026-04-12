-- +goose Up
CREATE TABLE episodes (
    id                       INTEGER PRIMARY KEY AUTOINCREMENT,
    series_id                INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    season_number            INTEGER NOT NULL,
    episode_number           INTEGER NOT NULL,
    absolute_episode_number  INTEGER,
    title                    TEXT NOT NULL DEFAULT '',
    overview                 TEXT NOT NULL DEFAULT '',
    air_date_utc             TEXT,
    monitored                INTEGER NOT NULL DEFAULT 1,
    episode_file_id          INTEGER,
    created_at               TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at               TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE (series_id, season_number, episode_number)
);

CREATE INDEX episodes_air_date_utc_idx ON episodes (air_date_utc);
CREATE INDEX episodes_episode_file_id_idx ON episodes (episode_file_id) WHERE episode_file_id IS NOT NULL;

-- +goose Down
DROP TABLE episodes;
