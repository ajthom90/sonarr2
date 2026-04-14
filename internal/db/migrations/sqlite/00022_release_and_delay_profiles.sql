-- SPDX-License-Identifier: GPL-3.0-or-later
-- Ported from Sonarr's ReleaseProfile and DelayProfile schemas
-- (src/NzbDrone.Core/Profiles/Releases/ReleaseProfile.cs,
--  src/NzbDrone.Core/Profiles/Delay/DelayProfile.cs).
-- Copyright (c) Team Sonarr, licensed under GPL-3.0.

-- +goose Up

-- Release Profiles: must-contain / must-not-contain term lists applied to
-- release titles during decision engine evaluation. Applied to a single
-- indexer (0 = any) and filtered by tags.
CREATE TABLE release_profiles (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    enabled    INTEGER NOT NULL DEFAULT 1,
    required   TEXT NOT NULL DEFAULT '[]',  -- JSON: string[]
    ignored    TEXT NOT NULL DEFAULT '[]',  -- JSON: string[]
    indexer_id INTEGER NOT NULL DEFAULT 0,
    tags       TEXT NOT NULL DEFAULT '[]'   -- JSON: int[]
);

-- Delay Profiles: ordered list of tag-scoped delay rules. The profile matching
-- a series's tags (or the default catch-all at Order=last) determines how long
-- to wait on each protocol before grabbing, allowing prefer-quality windows.
CREATE TABLE delay_profiles (
    id                                   INTEGER PRIMARY KEY AUTOINCREMENT,
    enable_usenet                        INTEGER NOT NULL DEFAULT 1,
    enable_torrent                       INTEGER NOT NULL DEFAULT 1,
    preferred_protocol                   TEXT NOT NULL DEFAULT 'usenet',  -- usenet|torrent
    usenet_delay                         INTEGER NOT NULL DEFAULT 0,       -- minutes
    torrent_delay                        INTEGER NOT NULL DEFAULT 0,       -- minutes
    sort_order                           INTEGER NOT NULL DEFAULT 2147483647,
    bypass_if_highest_quality            INTEGER NOT NULL DEFAULT 0,
    bypass_if_above_custom_format_score  INTEGER NOT NULL DEFAULT 0,
    minimum_custom_format_score          INTEGER NOT NULL DEFAULT 0,
    tags                                 TEXT NOT NULL DEFAULT '[]'        -- JSON: int[]
);

-- Seed the default catch-all delay profile (Order = MaxInt, no tags, no delay).
-- Matches Sonarr's built-in default so every series has at least one applicable profile.
INSERT INTO delay_profiles (
    enable_usenet, enable_torrent, preferred_protocol,
    usenet_delay, torrent_delay, sort_order
) VALUES (1, 1, 'usenet', 0, 0, 2147483647);

-- +goose Down
DROP TABLE delay_profiles;
DROP TABLE release_profiles;
