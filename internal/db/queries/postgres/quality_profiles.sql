-- name: CreateQualityProfile :one
INSERT INTO quality_profiles (name, upgrade_allowed, cutoff, items, min_format_score, cutoff_format_score, format_items)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, name, upgrade_allowed, cutoff, items, min_format_score, cutoff_format_score, format_items;

-- name: GetQualityProfileByID :one
SELECT id, name, upgrade_allowed, cutoff, items, min_format_score, cutoff_format_score, format_items
FROM quality_profiles
WHERE id = $1;

-- name: ListQualityProfiles :many
SELECT id, name, upgrade_allowed, cutoff, items, min_format_score, cutoff_format_score, format_items
FROM quality_profiles
ORDER BY name;

-- name: UpdateQualityProfile :exec
UPDATE quality_profiles
SET name                = $2,
    upgrade_allowed     = $3,
    cutoff              = $4,
    items               = $5,
    min_format_score    = $6,
    cutoff_format_score = $7,
    format_items        = $8
WHERE id = $1;

-- name: DeleteQualityProfile :exec
DELETE FROM quality_profiles WHERE id = $1;
