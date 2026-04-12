-- name: UpsertSeason :exec
INSERT INTO seasons (series_id, season_number, monitored)
VALUES (?, ?, ?)
ON CONFLICT (series_id, season_number) DO UPDATE
SET monitored = excluded.monitored;

-- name: GetSeason :one
SELECT series_id, season_number, monitored
FROM seasons
WHERE series_id = ? AND season_number = ?;

-- name: ListSeasonsForSeries :many
SELECT series_id, season_number, monitored
FROM seasons
WHERE series_id = ?
ORDER BY season_number;

-- name: DeleteSeason :exec
DELETE FROM seasons WHERE series_id = ? AND season_number = ?;
