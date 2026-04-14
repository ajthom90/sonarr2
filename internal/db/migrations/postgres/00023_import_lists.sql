-- SPDX-License-Identifier: GPL-3.0-or-later
-- Ported from Sonarr's ImportList + ImportListExclusion schemas.
-- Copyright (c) Team Sonarr, licensed under GPL-3.0.

-- +goose Up
CREATE TABLE import_lists (
    id                     SERIAL PRIMARY KEY,
    name                   TEXT NOT NULL,
    implementation         TEXT NOT NULL,
    settings               JSONB NOT NULL DEFAULT '{}'::jsonb,
    enable_automatic_add   BOOLEAN NOT NULL DEFAULT TRUE,
    should_monitor         TEXT NOT NULL DEFAULT 'all',
    should_monitor_existing BOOLEAN NOT NULL DEFAULT FALSE,
    should_search          BOOLEAN NOT NULL DEFAULT FALSE,
    root_folder_path       TEXT NOT NULL DEFAULT '',
    quality_profile_id     INTEGER NOT NULL DEFAULT 0,
    series_type            TEXT NOT NULL DEFAULT 'standard',
    season_folder          BOOLEAN NOT NULL DEFAULT TRUE,
    tags                   JSONB NOT NULL DEFAULT '[]'::jsonb,
    list_type              TEXT NOT NULL DEFAULT 'program',
    min_refresh_interval_mins INTEGER NOT NULL DEFAULT 60
);

CREATE TABLE import_list_exclusions (
    id      SERIAL PRIMARY KEY,
    tvdb_id INTEGER NOT NULL UNIQUE,
    title   TEXT NOT NULL
);

-- +goose Down
DROP TABLE import_list_exclusions;
DROP TABLE import_lists;
