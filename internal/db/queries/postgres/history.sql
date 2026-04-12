-- name: CreateHistoryEntry :one
INSERT INTO history (episode_id, series_id, source_title, quality_name, event_type, download_id, data)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, episode_id, series_id, source_title, quality_name, event_type, date, download_id, data;

-- name: ListForSeries :many
SELECT id, episode_id, series_id, source_title, quality_name, event_type, date, download_id, data
FROM history
WHERE series_id = $1
ORDER BY date DESC;

-- name: ListForEpisode :many
SELECT id, episode_id, series_id, source_title, quality_name, event_type, date, download_id, data
FROM history
WHERE episode_id = $1
ORDER BY date DESC;

-- name: FindByDownloadID :many
SELECT id, episode_id, series_id, source_title, quality_name, event_type, date, download_id, data
FROM history
WHERE download_id = $1
ORDER BY date DESC;

-- name: DeleteForSeries :exec
DELETE FROM history WHERE series_id = $1;
