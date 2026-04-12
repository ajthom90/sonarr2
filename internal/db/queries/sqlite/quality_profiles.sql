-- name: CreateQualityProfile :one
INSERT INTO quality_profiles (name, upgrade_allowed, cutoff, items, min_format_score, cutoff_format_score, format_items)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id, name, upgrade_allowed, cutoff, items, min_format_score, cutoff_format_score, format_items;

-- name: GetQualityProfileByID :one
SELECT id, name, upgrade_allowed, cutoff, items, min_format_score, cutoff_format_score, format_items
FROM quality_profiles
WHERE id = ?;

-- name: ListQualityProfiles :many
SELECT id, name, upgrade_allowed, cutoff, items, min_format_score, cutoff_format_score, format_items
FROM quality_profiles
ORDER BY name;

-- name: UpdateQualityProfile :exec
UPDATE quality_profiles
SET name                = ?,
    upgrade_allowed     = ?,
    cutoff              = ?,
    items               = ?,
    min_format_score    = ?,
    cutoff_format_score = ?,
    format_items        = ?
WHERE id = ?;

-- name: DeleteQualityProfile :exec
DELETE FROM quality_profiles WHERE id = ?;
