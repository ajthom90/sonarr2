-- +goose Up
CREATE TABLE series (
    id               BIGSERIAL PRIMARY KEY,
    tvdb_id          BIGINT NOT NULL UNIQUE,
    title            TEXT NOT NULL,
    slug             TEXT NOT NULL UNIQUE,
    status           TEXT NOT NULL DEFAULT 'continuing',
    series_type      TEXT NOT NULL DEFAULT 'standard',
    path             TEXT NOT NULL UNIQUE,
    monitored        BOOLEAN NOT NULL DEFAULT TRUE,
    added            TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX series_title_idx ON series (title);

-- +goose Down
DROP TABLE series;
