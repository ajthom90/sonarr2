-- name: CreateSeries :one
INSERT INTO series (tvdb_id, title, slug, status, series_type, path, monitored, quality_profile_id, season_folder, monitor_new_items)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, tvdb_id, title, slug, status, series_type, path, monitored, added, created_at, updated_at, quality_profile_id, season_folder, monitor_new_items;

-- name: GetSeries :one
SELECT id, tvdb_id, title, slug, status, series_type, path, monitored, added, created_at, updated_at, quality_profile_id, season_folder, monitor_new_items
FROM series
WHERE id = $1;

-- name: GetSeriesByTvdbID :one
SELECT id, tvdb_id, title, slug, status, series_type, path, monitored, added, created_at, updated_at, quality_profile_id, season_folder, monitor_new_items
FROM series
WHERE tvdb_id = $1;

-- name: GetSeriesBySlug :one
SELECT id, tvdb_id, title, slug, status, series_type, path, monitored, added, created_at, updated_at, quality_profile_id, season_folder, monitor_new_items
FROM series
WHERE slug = $1;

-- name: ListSeries :many
SELECT id, tvdb_id, title, slug, status, series_type, path, monitored, added, created_at, updated_at, quality_profile_id, season_folder, monitor_new_items
FROM series
ORDER BY title;

-- name: UpdateSeries :exec
UPDATE series
SET tvdb_id = $2,
    title = $3,
    slug = $4,
    status = $5,
    series_type = $6,
    path = $7,
    monitored = $8,
    quality_profile_id = $9,
    season_folder = $10,
    monitor_new_items = $11,
    updated_at = now()
WHERE id = $1;

-- name: DeleteSeries :exec
DELETE FROM series WHERE id = $1;
