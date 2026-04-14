-- name: CreateTag :one
INSERT INTO tags (label)
VALUES ($1)
RETURNING id, label;

-- name: GetTagByID :one
SELECT id, label FROM tags WHERE id = $1;

-- name: GetTagByLabel :one
SELECT id, label FROM tags WHERE label = $1;

-- name: ListTags :many
SELECT id, label FROM tags ORDER BY label;

-- name: UpdateTag :exec
UPDATE tags SET label = $1 WHERE id = $2;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = $1;
