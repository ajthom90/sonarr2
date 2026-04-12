-- +goose Up
CREATE TABLE history (
    id            BIGSERIAL PRIMARY KEY,
    episode_id    BIGINT NOT NULL,
    series_id     BIGINT NOT NULL,
    source_title  TEXT NOT NULL,
    quality_name  TEXT NOT NULL DEFAULT '',
    event_type    TEXT NOT NULL,
    date          TIMESTAMPTZ NOT NULL DEFAULT now(),
    download_id   TEXT NOT NULL DEFAULT '',
    data          JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX history_series_date_idx ON history (series_id, date DESC);
CREATE INDEX history_episode_date_idx ON history (episode_id, date DESC);
CREATE INDEX history_download_id_idx ON history (download_id) WHERE download_id != '';

-- +goose Down
DROP TABLE history;
