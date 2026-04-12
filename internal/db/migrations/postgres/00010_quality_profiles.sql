-- +goose Up
CREATE TABLE quality_profiles (
    id                  SERIAL PRIMARY KEY,
    name                TEXT NOT NULL,
    upgrade_allowed     BOOLEAN NOT NULL DEFAULT TRUE,
    cutoff              INTEGER NOT NULL DEFAULT 0,
    items               JSONB NOT NULL DEFAULT '[]',
    min_format_score    INTEGER NOT NULL DEFAULT 0,
    cutoff_format_score INTEGER NOT NULL DEFAULT 0,
    format_items        JSONB NOT NULL DEFAULT '[]'
);

-- +goose Down
DROP TABLE quality_profiles;
