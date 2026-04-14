-- SPDX-License-Identifier: GPL-3.0-or-later
-- Ported from Sonarr's Blocklist schema
-- (src/NzbDrone.Core/Blocklisting/Blocklist.cs and its EF migration).
-- Copyright (c) Team Sonarr, licensed under GPL-3.0.

-- +goose Up
CREATE TABLE blocklist (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    series_id          INTEGER NOT NULL,
    episode_ids        TEXT NOT NULL DEFAULT '[]', -- JSON array of ints
    source_title       TEXT NOT NULL,
    quality            TEXT NOT NULL DEFAULT '{}', -- QualityModel JSON
    languages          TEXT NOT NULL DEFAULT '[]', -- Language[] JSON
    date               TIMESTAMP NOT NULL,
    published_date     TIMESTAMP,
    size               INTEGER,
    protocol           TEXT NOT NULL DEFAULT '',   -- "usenet" | "torrent"
    indexer            TEXT NOT NULL DEFAULT '',
    indexer_flags      INTEGER NOT NULL DEFAULT 0,
    release_type       TEXT NOT NULL DEFAULT '',
    message            TEXT NOT NULL DEFAULT '',
    torrent_info_hash  TEXT
);

CREATE INDEX idx_blocklist_series_id ON blocklist(series_id);
CREATE INDEX idx_blocklist_date      ON blocklist(date DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_blocklist_date;
DROP INDEX IF EXISTS idx_blocklist_series_id;
DROP TABLE blocklist;
