-- name: CreateSeries :one
INSERT INTO series (tvdb_id, title, slug, status, series_type, path, monitored, quality_profile_id, season_folder, monitor_new_items)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, tvdb_id, title, slug, status, series_type, path, monitored, added, created_at, updated_at, quality_profile_id, season_folder, monitor_new_items;

-- name: GetSeries :one
SELECT id, tvdb_id, title, slug, status, series_type, path, monitored, added, created_at, updated_at, quality_profile_id, season_folder, monitor_new_items
FROM series
WHERE id = ?;

-- name: GetSeriesByTvdbID :one
SELECT id, tvdb_id, title, slug, status, series_type, path, monitored, added, created_at, updated_at, quality_profile_id, season_folder, monitor_new_items
FROM series
WHERE tvdb_id = ?;

-- name: GetSeriesBySlug :one
SELECT id, tvdb_id, title, slug, status, series_type, path, monitored, added, created_at, updated_at, quality_profile_id, season_folder, monitor_new_items
FROM series
WHERE slug = ?;

-- name: ListSeries :many
SELECT id, tvdb_id, title, slug, status, series_type, path, monitored, added, created_at, updated_at, quality_profile_id, season_folder, monitor_new_items
FROM series
ORDER BY title;

-- name: UpdateSeries :exec
UPDATE series
SET tvdb_id = ?,
    title = ?,
    slug = ?,
    status = ?,
    series_type = ?,
    path = ?,
    monitored = ?,
    quality_profile_id = ?,
    season_folder = ?,
    monitor_new_items = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteSeries :exec
DELETE FROM series WHERE id = ?;
