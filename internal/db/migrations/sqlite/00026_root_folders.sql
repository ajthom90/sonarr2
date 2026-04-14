-- +goose Up
CREATE TABLE root_folders (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    path       TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX root_folders_path_idx ON root_folders (path);

-- +goose Down
DROP TABLE root_folders;
