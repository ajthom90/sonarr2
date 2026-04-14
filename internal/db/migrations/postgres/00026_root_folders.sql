-- +goose Up
CREATE TABLE root_folders (
    id         SERIAL PRIMARY KEY,
    path       TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX root_folders_path_idx ON root_folders (path);

-- +goose Down
DROP TABLE root_folders;
