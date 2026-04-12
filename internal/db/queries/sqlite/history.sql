-- name: CreateHistoryEntry :one
INSERT INTO history (episode_id, series_id, source_title, quality_name, event_type, download_id, data)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id, episode_id, series_id, source_title, quality_name, event_type, date, download_id, data;

-- name: ListForSeries :many
SELECT id, episode_id, series_id, source_title, quality_name, event_type, date, download_id, data
FROM history
WHERE series_id = ?
ORDER BY date DESC;

-- name: ListForEpisode :many
SELECT id, episode_id, series_id, source_title, quality_name, event_type, date, download_id, data
FROM history
WHERE episode_id = ?
ORDER BY date DESC;

-- name: FindByDownloadID :many
SELECT id, episode_id, series_id, source_title, quality_name, event_type, date, download_id, data
FROM history
WHERE download_id = ?
ORDER BY date DESC;

-- name: DeleteForSeries :exec
DELETE FROM history WHERE series_id = ?;
