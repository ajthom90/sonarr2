-- name: GetAllQualityDefinitions :many
SELECT id, name, source, resolution, min_size, max_size, preferred_size
FROM quality_definitions
ORDER BY id;

-- name: GetQualityDefinitionByID :one
SELECT id, name, source, resolution, min_size, max_size, preferred_size
FROM quality_definitions
WHERE id = $1;

-- name: UpdateQualityDefinitionSizes :exec
UPDATE quality_definitions
SET min_size       = $2,
    max_size       = $3,
    preferred_size = $4
WHERE id = $1;
