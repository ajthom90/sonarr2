-- SPDX-License-Identifier: GPL-3.0-or-later
-- Ported from Sonarr's Blocklist schema
-- (src/NzbDrone.Core/Blocklisting/Blocklist.cs and its EF migration).
-- Copyright (c) Team Sonarr, licensed under GPL-3.0.

-- +goose Up
CREATE TABLE blocklist (
    id                 SERIAL PRIMARY KEY,
    series_id          INTEGER NOT NULL,
    episode_ids        JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_title       TEXT NOT NULL,
    quality            JSONB NOT NULL DEFAULT '{}'::jsonb,
    languages          JSONB NOT NULL DEFAULT '[]'::jsonb,
    date               TIMESTAMPTZ NOT NULL,
    published_date     TIMESTAMPTZ,
    size               BIGINT,
    protocol           TEXT NOT NULL DEFAULT '',
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
