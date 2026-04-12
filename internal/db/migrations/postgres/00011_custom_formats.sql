-- +goose Up
CREATE TABLE custom_formats (
    id                    SERIAL PRIMARY KEY,
    name                  TEXT NOT NULL,
    include_when_renaming BOOLEAN NOT NULL DEFAULT FALSE,
    specifications        JSONB NOT NULL DEFAULT '[]'
);

-- +goose Down
DROP TABLE custom_formats;
