-- name: CreateCustomFormat :one
INSERT INTO custom_formats (name, include_when_renaming, specifications)
VALUES (?, ?, ?)
RETURNING id, name, include_when_renaming, specifications;

-- name: GetCustomFormatByID :one
SELECT id, name, include_when_renaming, specifications
FROM custom_formats
WHERE id = ?;

-- name: ListCustomFormats :many
SELECT id, name, include_when_renaming, specifications
FROM custom_formats
ORDER BY name;

-- name: UpdateCustomFormat :exec
UPDATE custom_formats
SET name                  = ?,
    include_when_renaming = ?,
    specifications        = ?
WHERE id = ?;

-- name: DeleteCustomFormat :exec
DELETE FROM custom_formats WHERE id = ?;
