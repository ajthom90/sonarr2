-- name: CreateCustomFormat :one
INSERT INTO custom_formats (name, include_when_renaming, specifications)
VALUES ($1, $2, $3)
RETURNING id, name, include_when_renaming, specifications;

-- name: GetCustomFormatByID :one
SELECT id, name, include_when_renaming, specifications
FROM custom_formats
WHERE id = $1;

-- name: ListCustomFormats :many
SELECT id, name, include_when_renaming, specifications
FROM custom_formats
ORDER BY name;

-- name: UpdateCustomFormat :exec
UPDATE custom_formats
SET name                  = $2,
    include_when_renaming = $3,
    specifications        = $4
WHERE id = $1;

-- name: DeleteCustomFormat :exec
DELETE FROM custom_formats WHERE id = $1;
