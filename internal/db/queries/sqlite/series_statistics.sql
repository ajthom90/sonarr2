-- name: UpsertSeriesStatistics :exec
INSERT INTO series_statistics (
    series_id, episode_count, episode_file_count, monitored_episode_count, size_on_disk
)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (series_id) DO UPDATE
SET episode_count = excluded.episode_count,
    episode_file_count = excluded.episode_file_count,
    monitored_episode_count = excluded.monitored_episode_count,
    size_on_disk = excluded.size_on_disk,
    updated_at = datetime('now');

-- name: GetSeriesStatistics :one
SELECT series_id, episode_count, episode_file_count, monitored_episode_count,
       size_on_disk, updated_at
FROM series_statistics
WHERE series_id = ?;

-- name: DeleteSeriesStatistics :exec
DELETE FROM series_statistics WHERE series_id = ?;
