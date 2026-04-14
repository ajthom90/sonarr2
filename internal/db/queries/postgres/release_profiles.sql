-- name: CreateReleaseProfile :one
INSERT INTO release_profiles (name, enabled, required, ignored, indexer_id, tags)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, enabled, required, ignored, indexer_id, tags;

-- name: GetReleaseProfileByID :one
SELECT id, name, enabled, required, ignored, indexer_id, tags
FROM release_profiles WHERE id = $1;

-- name: ListReleaseProfiles :many
SELECT id, name, enabled, required, ignored, indexer_id, tags
FROM release_profiles ORDER BY name;

-- name: UpdateReleaseProfile :exec
UPDATE release_profiles
SET name = $1, enabled = $2, required = $3, ignored = $4, indexer_id = $5, tags = $6
WHERE id = $7;

-- name: DeleteReleaseProfile :exec
DELETE FROM release_profiles WHERE id = $1;
