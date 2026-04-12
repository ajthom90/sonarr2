-- +goose Up
CREATE TABLE series_statistics (
    series_id                 BIGINT PRIMARY KEY REFERENCES series(id) ON DELETE CASCADE,
    episode_count             INTEGER NOT NULL DEFAULT 0,
    episode_file_count        INTEGER NOT NULL DEFAULT 0,
    monitored_episode_count   INTEGER NOT NULL DEFAULT 0,
    size_on_disk              BIGINT NOT NULL DEFAULT 0,
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE series_statistics;
