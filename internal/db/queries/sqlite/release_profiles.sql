-- name: CreateReleaseProfile :one
INSERT INTO release_profiles (name, enabled, required, ignored, indexer_id, tags)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, name, enabled, required, ignored, indexer_id, tags;

-- name: GetReleaseProfileByID :one
SELECT id, name, enabled, required, ignored, indexer_id, tags
FROM release_profiles WHERE id = ?;

-- name: ListReleaseProfiles :many
SELECT id, name, enabled, required, ignored, indexer_id, tags
FROM release_profiles ORDER BY name;

-- name: UpdateReleaseProfile :exec
UPDATE release_profiles
SET name = ?, enabled = ?, required = ?, ignored = ?, indexer_id = ?, tags = ?
WHERE id = ?;

-- name: DeleteReleaseProfile :exec
DELETE FROM release_profiles WHERE id = ?;
