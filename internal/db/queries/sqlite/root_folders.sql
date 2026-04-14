-- name: CreateRootFolder :one
INSERT INTO root_folders (path)
VALUES (?)
RETURNING id, path, created_at;

-- name: GetRootFolder :one
SELECT id, path, created_at
FROM root_folders
WHERE id = ?;

-- name: GetRootFolderByPath :one
SELECT id, path, created_at
FROM root_folders
WHERE path = ?;

-- name: ListRootFolders :many
SELECT id, path, created_at
FROM root_folders
ORDER BY path;

-- name: DeleteRootFolder :exec
DELETE FROM root_folders WHERE id = ?;
