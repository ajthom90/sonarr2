-- name: CreateSeries :one
INSERT INTO series (tvdb_id, title, slug, status, series_type, path, monitored)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id, tvdb_id, title, slug, status, series_type, path, monitored, added, created_at, updated_at;

-- name: GetSeries :one
SELECT id, tvdb_id, title, slug, status, series_type, path, monitored, added, created_at, updated_at
FROM series
WHERE id = ?;

-- name: GetSeriesByTvdbID :one
SELECT id, tvdb_id, title, slug, status, series_type, path, monitored, added, created_at, updated_at
FROM series
WHERE tvdb_id = ?;

-- name: GetSeriesBySlug :one
SELECT id, tvdb_id, title, slug, status, series_type, path, monitored, added, created_at, updated_at
FROM series
WHERE slug = ?;

-- name: ListSeries :many
SELECT id, tvdb_id, title, slug, status, series_type, path, monitored, added, created_at, updated_at
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
    updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteSeries :exec
DELETE FROM series WHERE id = ?;
