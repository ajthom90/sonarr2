-- name: CreateRootFolder :one
INSERT INTO root_folders (path)
VALUES ($1)
RETURNING id, path, created_at;

-- name: GetRootFolder :one
SELECT id, path, created_at
FROM root_folders
WHERE id = $1;

-- name: GetRootFolderByPath :one
SELECT id, path, created_at
FROM root_folders
WHERE path = $1;

-- name: ListRootFolders :many
SELECT id, path, created_at
FROM root_folders
ORDER BY path;

-- name: DeleteRootFolder :exec
DELETE FROM root_folders WHERE id = $1;
