-- SPDX-License-Identifier: GPL-3.0-or-later
-- Ported from Sonarr's ImportList + ImportListExclusion schemas
-- (src/NzbDrone.Core/ImportLists/).
-- Copyright (c) Team Sonarr, licensed under GPL-3.0.

-- +goose Up

CREATE TABLE import_lists (
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    name                   TEXT NOT NULL,
    implementation         TEXT NOT NULL,
    settings               TEXT NOT NULL DEFAULT '{}',
    enable_automatic_add   INTEGER NOT NULL DEFAULT 1,
    should_monitor         TEXT NOT NULL DEFAULT 'all',   -- all|future|missing|existing|pilot|firstSeason|latestSeason|none
    should_monitor_existing INTEGER NOT NULL DEFAULT 0,
    should_search          INTEGER NOT NULL DEFAULT 0,
    root_folder_path       TEXT NOT NULL DEFAULT '',
    quality_profile_id     INTEGER NOT NULL DEFAULT 0,
    series_type            TEXT NOT NULL DEFAULT 'standard',
    season_folder          INTEGER NOT NULL DEFAULT 1,
    tags                   TEXT NOT NULL DEFAULT '[]',    -- JSON: int[]
    list_type              TEXT NOT NULL DEFAULT 'program', -- program|advanced|other
    min_refresh_interval_mins INTEGER NOT NULL DEFAULT 60
);

-- Exclusions prevent auto-add for specific TVDB IDs. User-managed from the
-- /settings/importlists → Exclusions sub-panel.
CREATE TABLE import_list_exclusions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    tvdb_id    INTEGER NOT NULL UNIQUE,
    title      TEXT NOT NULL
);

-- +goose Down
DROP TABLE import_list_exclusions;
DROP TABLE import_lists;
