-- +goose Up
CREATE TABLE seasons (
    series_id     INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL,
    monitored     INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (series_id, season_number)
);

-- +goose Down
DROP TABLE seasons;
