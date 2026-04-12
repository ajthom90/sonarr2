-- +goose Up
CREATE TABLE seasons (
    series_id     BIGINT NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL,
    monitored     BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY (series_id, season_number)
);

-- +goose Down
DROP TABLE seasons;
