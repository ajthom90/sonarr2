-- SPDX-License-Identifier: GPL-3.0-or-later
-- Ported from Sonarr's ReleaseProfile and DelayProfile schemas.
-- Copyright (c) Team Sonarr, licensed under GPL-3.0.

-- +goose Up
CREATE TABLE release_profiles (
    id         SERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    required   JSONB NOT NULL DEFAULT '[]'::jsonb,
    ignored    JSONB NOT NULL DEFAULT '[]'::jsonb,
    indexer_id INTEGER NOT NULL DEFAULT 0,
    tags       JSONB NOT NULL DEFAULT '[]'::jsonb
);

CREATE TABLE delay_profiles (
    id                                   SERIAL PRIMARY KEY,
    enable_usenet                        BOOLEAN NOT NULL DEFAULT TRUE,
    enable_torrent                       BOOLEAN NOT NULL DEFAULT TRUE,
    preferred_protocol                   TEXT NOT NULL DEFAULT 'usenet',
    usenet_delay                         INTEGER NOT NULL DEFAULT 0,
    torrent_delay                        INTEGER NOT NULL DEFAULT 0,
    sort_order                           INTEGER NOT NULL DEFAULT 2147483647,
    bypass_if_highest_quality            BOOLEAN NOT NULL DEFAULT FALSE,
    bypass_if_above_custom_format_score  BOOLEAN NOT NULL DEFAULT FALSE,
    minimum_custom_format_score          INTEGER NOT NULL DEFAULT 0,
    tags                                 JSONB NOT NULL DEFAULT '[]'::jsonb
);

-- Seed default catch-all delay profile.
INSERT INTO delay_profiles (
    enable_usenet, enable_torrent, preferred_protocol,
    usenet_delay, torrent_delay, sort_order
) VALUES (TRUE, TRUE, 'usenet', 0, 0, 2147483647);

-- +goose Down
DROP TABLE delay_profiles;
DROP TABLE release_profiles;
