-- +goose Up
CREATE TABLE episodes (
    id                       BIGSERIAL PRIMARY KEY,
    series_id                BIGINT NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    season_number            INTEGER NOT NULL,
    episode_number           INTEGER NOT NULL,
    absolute_episode_number  INTEGER,
    title                    TEXT NOT NULL DEFAULT '',
    overview                 TEXT NOT NULL DEFAULT '',
    air_date_utc             TIMESTAMPTZ,
    monitored                BOOLEAN NOT NULL DEFAULT TRUE,
    episode_file_id          BIGINT,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (series_id, season_number, episode_number)
);

CREATE INDEX episodes_air_date_utc_idx ON episodes (air_date_utc);
CREATE INDEX episodes_episode_file_id_idx ON episodes (episode_file_id) WHERE episode_file_id IS NOT NULL;

-- +goose Down
DROP TABLE episodes;
