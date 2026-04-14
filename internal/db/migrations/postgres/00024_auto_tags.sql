-- SPDX-License-Identifier: GPL-3.0-or-later

-- +goose Up
CREATE TABLE auto_tags (
    id                       SERIAL PRIMARY KEY,
    name                     TEXT NOT NULL UNIQUE,
    remove_tags_automatically BOOLEAN NOT NULL DEFAULT FALSE,
    tags                     JSONB NOT NULL DEFAULT '[]'::jsonb,
    specifications           JSONB NOT NULL DEFAULT '[]'::jsonb
);

-- +goose Down
DROP TABLE auto_tags;
