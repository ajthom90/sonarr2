-- SPDX-License-Identifier: GPL-3.0-or-later
-- Ported from Sonarr's SceneMapping schema
-- (src/NzbDrone.Core/DataAugmentation/Scene/SceneMapping.cs).

-- +goose Up
CREATE TABLE scene_mappings (
    id                       INTEGER PRIMARY KEY AUTOINCREMENT,
    tvdb_id                  INTEGER NOT NULL,
    season_number            INTEGER,              -- NULL = all seasons
    scene_season_number      INTEGER,
    scene_origin             TEXT NOT NULL DEFAULT '',
    comment                  TEXT NOT NULL DEFAULT '',
    filter_regex             TEXT NOT NULL DEFAULT '',
    parse_term               TEXT NOT NULL DEFAULT '',
    search_term              TEXT NOT NULL DEFAULT '',
    title                    TEXT NOT NULL DEFAULT '',
    type                     TEXT NOT NULL DEFAULT '',
    updated_at               TIMESTAMP NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_scene_mappings_tvdb ON scene_mappings(tvdb_id);

-- +goose Down
DROP INDEX IF EXISTS idx_scene_mappings_tvdb;
DROP TABLE scene_mappings;
