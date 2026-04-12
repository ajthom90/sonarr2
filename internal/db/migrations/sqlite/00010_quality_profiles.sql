-- +goose Up
CREATE TABLE quality_profiles (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    name                TEXT NOT NULL,
    upgrade_allowed     INTEGER NOT NULL DEFAULT 1,
    cutoff              INTEGER NOT NULL DEFAULT 0,
    items               TEXT NOT NULL DEFAULT '[]',
    min_format_score    INTEGER NOT NULL DEFAULT 0,
    cutoff_format_score INTEGER NOT NULL DEFAULT 0,
    format_items        TEXT NOT NULL DEFAULT '[]'
);

-- +goose Down
DROP TABLE quality_profiles;
