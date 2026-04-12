-- +goose Up
CREATE TABLE custom_formats (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    name                  TEXT NOT NULL,
    include_when_renaming INTEGER NOT NULL DEFAULT 0,
    specifications        TEXT NOT NULL DEFAULT '[]'
);

-- +goose Down
DROP TABLE custom_formats;
