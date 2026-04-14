-- SPDX-License-Identifier: GPL-3.0-or-later
-- Ported from Sonarr's AutoTag schema (src/NzbDrone.Core/AutoTagging/).

-- +goose Up
CREATE TABLE auto_tags (
    id                       INTEGER PRIMARY KEY AUTOINCREMENT,
    name                     TEXT NOT NULL UNIQUE,
    remove_tags_automatically INTEGER NOT NULL DEFAULT 0,
    tags                     TEXT NOT NULL DEFAULT '[]',   -- JSON int[]
    specifications           TEXT NOT NULL DEFAULT '[]'    -- JSON: [{implementation, negate, required, fields:[{name,value}]}]
);

-- +goose Down
DROP TABLE auto_tags;
