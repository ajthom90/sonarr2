-- +goose Up
CREATE TABLE indexers (
    id                        SERIAL PRIMARY KEY,
    name                      TEXT NOT NULL,
    implementation            TEXT NOT NULL,
    settings                  JSONB NOT NULL DEFAULT '{}',
    enable_rss                BOOLEAN NOT NULL DEFAULT TRUE,
    enable_automatic_search   BOOLEAN NOT NULL DEFAULT TRUE,
    enable_interactive_search BOOLEAN NOT NULL DEFAULT TRUE,
    priority                  INTEGER NOT NULL DEFAULT 25,
    added                     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE indexers;
