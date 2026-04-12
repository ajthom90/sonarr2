-- +goose Up
CREATE TABLE history (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    episode_id    INTEGER NOT NULL,
    series_id     INTEGER NOT NULL,
    source_title  TEXT NOT NULL,
    quality_name  TEXT NOT NULL DEFAULT '',
    event_type    TEXT NOT NULL,
    date          TEXT NOT NULL DEFAULT (datetime('now')),
    download_id   TEXT NOT NULL DEFAULT '',
    data          TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX history_series_date_idx ON history (series_id, date DESC);
CREATE INDEX history_episode_date_idx ON history (episode_id, date DESC);
CREATE INDEX history_download_id_idx ON history (download_id) WHERE download_id != '';

-- +goose Down
DROP TABLE history;
