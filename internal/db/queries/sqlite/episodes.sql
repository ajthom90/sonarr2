-- name: CreateEpisode :one
INSERT INTO episodes (
    series_id, season_number, episode_number, absolute_episode_number,
    title, overview, air_date_utc, monitored, episode_file_id
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, series_id, season_number, episode_number, absolute_episode_number,
          title, overview, air_date_utc, monitored, episode_file_id,
          created_at, updated_at;

-- name: GetEpisode :one
SELECT id, series_id, season_number, episode_number, absolute_episode_number,
       title, overview, air_date_utc, monitored, episode_file_id,
       created_at, updated_at
FROM episodes
WHERE id = ?;

-- name: ListEpisodesForSeries :many
SELECT id, series_id, season_number, episode_number, absolute_episode_number,
       title, overview, air_date_utc, monitored, episode_file_id,
       created_at, updated_at
FROM episodes
WHERE series_id = ?
ORDER BY season_number, episode_number;

-- name: UpdateEpisode :exec
UPDATE episodes
SET absolute_episode_number = ?,
    title = ?,
    overview = ?,
    air_date_utc = ?,
    monitored = ?,
    episode_file_id = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: SetEpisodeMonitored :exec
UPDATE episodes
SET monitored = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteEpisode :exec
DELETE FROM episodes WHERE id = ?;

-- name: CountEpisodesForSeries :one
SELECT
    COUNT(*) AS episode_count,
    SUM(CASE WHEN monitored = 1 THEN 1 ELSE 0 END) AS monitored_count
FROM episodes
WHERE series_id = ?;
