-- +goose Up
CREATE TABLE quality_definitions (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    name           TEXT NOT NULL UNIQUE,
    source         TEXT NOT NULL,
    resolution     TEXT NOT NULL,
    min_size       REAL NOT NULL DEFAULT 0,
    max_size       REAL NOT NULL DEFAULT 0,
    preferred_size REAL NOT NULL DEFAULT 0
);

-- Seed the standard quality definitions.
INSERT INTO quality_definitions (name, source, resolution, min_size, max_size, preferred_size) VALUES
    ('SDTV', 'television', '480p', 0.5, 100, 50),
    ('WEBDL-480p', 'webdl', '480p', 0.5, 100, 50),
    ('WEBRip-480p', 'webrip', '480p', 0.5, 100, 50),
    ('DVD', 'dvd', '480p', 0.5, 100, 50),
    ('HDTV-720p', 'television', '720p', 3, 200, 95),
    ('WEBDL-720p', 'webdl', '720p', 3, 200, 95),
    ('WEBRip-720p', 'webrip', '720p', 3, 200, 95),
    ('Bluray-720p', 'bluray', '720p', 3, 200, 95),
    ('HDTV-1080p', 'television', '1080p', 3, 400, 190),
    ('WEBDL-1080p', 'webdl', '1080p', 3, 400, 190),
    ('WEBRip-1080p', 'webrip', '1080p', 3, 400, 190),
    ('Bluray-1080p', 'bluray', '1080p', 3, 400, 190),
    ('Bluray-1080p Remux', 'remux', '1080p', 10, 600, 400),
    ('HDTV-2160p', 'television', '2160p', 10, 800, 400),
    ('WEBDL-2160p', 'webdl', '2160p', 10, 800, 400),
    ('WEBRip-2160p', 'webrip', '2160p', 10, 800, 400),
    ('Bluray-2160p', 'bluray', '2160p', 10, 800, 400),
    ('Bluray-2160p Remux', 'remux', '2160p', 20, 1200, 800);

-- +goose Down
DROP TABLE quality_definitions;
