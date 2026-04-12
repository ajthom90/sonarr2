-- name: CreateEpisode :one
INSERT INTO episodes (
    series_id, season_number, episode_number, absolute_episode_number,
    title, overview, air_date_utc, monitored, episode_file_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, series_id, season_number, episode_number, absolute_episode_number,
          title, overview, air_date_utc, monitored, episode_file_id,
          created_at, updated_at;

-- name: GetEpisode :one
SELECT id, series_id, season_number, episode_number, absolute_episode_number,
       title, overview, air_date_utc, monitored, episode_file_id,
       created_at, updated_at
FROM episodes
WHERE id = $1;

-- name: ListEpisodesForSeries :many
SELECT id, series_id, season_number, episode_number, absolute_episode_number,
       title, overview, air_date_utc, monitored, episode_file_id,
       created_at, updated_at
FROM episodes
WHERE series_id = $1
ORDER BY season_number, episode_number;

-- name: UpdateEpisode :exec
UPDATE episodes
SET absolute_episode_number = $2,
    title = $3,
    overview = $4,
    air_date_utc = $5,
    monitored = $6,
    episode_file_id = $7,
    updated_at = now()
WHERE id = $1;

-- name: DeleteEpisode :exec
DELETE FROM episodes WHERE id = $1;

-- name: CountEpisodesForSeries :one
SELECT
    COUNT(*) AS episode_count,
    COUNT(*) FILTER (WHERE monitored = TRUE) AS monitored_count
FROM episodes
WHERE series_id = $1;
