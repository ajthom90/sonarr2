-- name: UpsertSeriesStatistics :exec
INSERT INTO series_statistics (
    series_id, episode_count, episode_file_count, monitored_episode_count, size_on_disk
)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (series_id) DO UPDATE
SET episode_count = EXCLUDED.episode_count,
    episode_file_count = EXCLUDED.episode_file_count,
    monitored_episode_count = EXCLUDED.monitored_episode_count,
    size_on_disk = EXCLUDED.size_on_disk,
    updated_at = now();

-- name: GetSeriesStatistics :one
SELECT series_id, episode_count, episode_file_count, monitored_episode_count,
       size_on_disk, updated_at
FROM series_statistics
WHERE series_id = $1;

-- name: DeleteSeriesStatistics :exec
DELETE FROM series_statistics WHERE series_id = $1;
