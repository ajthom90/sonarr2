-- SPDX-License-Identifier: GPL-3.0-or-later
-- Ported from Sonarr's tag table schema
-- (src/NzbDrone.Core/Datastore/Migration/*_*.cs, src/NzbDrone.Core/Tags/Tag.cs).
-- Copyright (c) Team Sonarr, licensed under GPL-3.0.

-- +goose Up
CREATE TABLE tags (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    label TEXT NOT NULL UNIQUE
);

-- +goose Down
DROP TABLE tags;
