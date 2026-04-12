-- +goose Up
CREATE TABLE indexers (
    id                        INTEGER PRIMARY KEY AUTOINCREMENT,
    name                      TEXT NOT NULL,
    implementation            TEXT NOT NULL,
    settings                  TEXT NOT NULL DEFAULT '{}',
    enable_rss                INTEGER NOT NULL DEFAULT 1,
    enable_automatic_search   INTEGER NOT NULL DEFAULT 1,
    enable_interactive_search INTEGER NOT NULL DEFAULT 1,
    priority                  INTEGER NOT NULL DEFAULT 25,
    added                     TEXT NOT NULL DEFAULT (datetime('now'))
);

-- +goose Down
DROP TABLE indexers;
