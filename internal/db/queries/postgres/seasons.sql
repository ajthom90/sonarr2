-- name: UpsertSeason :exec
INSERT INTO seasons (series_id, season_number, monitored)
VALUES ($1, $2, $3)
ON CONFLICT (series_id, season_number) DO UPDATE
SET monitored = EXCLUDED.monitored;

-- name: GetSeason :one
SELECT series_id, season_number, monitored
FROM seasons
WHERE series_id = $1 AND season_number = $2;

-- name: ListSeasonsForSeries :many
SELECT series_id, season_number, monitored
FROM seasons
WHERE series_id = $1
ORDER BY season_number;

-- name: DeleteSeason :exec
DELETE FROM seasons WHERE series_id = $1 AND season_number = $2;
