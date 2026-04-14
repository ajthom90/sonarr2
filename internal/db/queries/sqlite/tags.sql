-- name: CreateTag :one
INSERT INTO tags (label)
VALUES (?)
RETURNING id, label;

-- name: GetTagByID :one
SELECT id, label FROM tags WHERE id = ?;

-- name: GetTagByLabel :one
SELECT id, label FROM tags WHERE label = ?;

-- name: ListTags :many
SELECT id, label FROM tags ORDER BY label;

-- name: UpdateTag :exec
UPDATE tags SET label = ? WHERE id = ?;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = ?;
