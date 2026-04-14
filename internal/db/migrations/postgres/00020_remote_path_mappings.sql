-- SPDX-License-Identifier: GPL-3.0-or-later
-- Ported from Sonarr's RemotePathMapping schema
-- (src/NzbDrone.Core/RemotePathMappings/RemotePathMapping.cs).
-- Copyright (c) Team Sonarr, licensed under GPL-3.0.

-- +goose Up
CREATE TABLE remote_path_mappings (
    id          SERIAL PRIMARY KEY,
    host        TEXT NOT NULL,
    remote_path TEXT NOT NULL,
    local_path  TEXT NOT NULL
);

CREATE INDEX idx_remote_path_mappings_host ON remote_path_mappings(host);

-- +goose Down
DROP INDEX IF EXISTS idx_remote_path_mappings_host;
DROP TABLE remote_path_mappings;
