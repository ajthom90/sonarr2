-- name: CreateEpisodeFile :one
INSERT INTO episode_files (
    series_id, season_number, relative_path, size, release_group, quality_name
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, series_id, season_number, relative_path, size, date_added,
          release_group, quality_name, created_at, updated_at;

-- name: GetEpisodeFile :one
SELECT id, series_id, season_number, relative_path, size, date_added,
       release_group, quality_name, created_at, updated_at
FROM episode_files
WHERE id = $1;

-- name: ListEpisodeFilesForSeries :many
SELECT id, series_id, season_number, relative_path, size, date_added,
       release_group, quality_name, created_at, updated_at
FROM episode_files
WHERE series_id = $1
ORDER BY season_number, relative_path;

-- name: DeleteEpisodeFile :exec
DELETE FROM episode_files WHERE id = $1;

-- name: SumEpisodeFileSizesForSeries :one
SELECT
    COUNT(*) AS file_count,
    COALESCE(SUM(size), 0)::BIGINT AS size_on_disk
FROM episode_files
WHERE series_id = $1;
