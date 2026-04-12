-- +goose Up
CREATE TABLE episode_files (
    id             BIGSERIAL PRIMARY KEY,
    series_id      BIGINT NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    season_number  INTEGER NOT NULL,
    relative_path  TEXT NOT NULL,
    size           BIGINT NOT NULL,
    date_added     TIMESTAMPTZ NOT NULL DEFAULT now(),
    release_group  TEXT NOT NULL DEFAULT '',
    quality_name   TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX episode_files_series_season_idx ON episode_files (series_id, season_number);

-- +goose Down
DROP TABLE episode_files;
