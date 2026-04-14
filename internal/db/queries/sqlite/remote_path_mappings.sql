-- name: CreateRemotePathMapping :one
INSERT INTO remote_path_mappings (host, remote_path, local_path)
VALUES (?, ?, ?)
RETURNING id, host, remote_path, local_path;

-- name: GetRemotePathMappingByID :one
SELECT id, host, remote_path, local_path
FROM remote_path_mappings WHERE id = ?;

-- name: ListRemotePathMappings :many
SELECT id, host, remote_path, local_path
FROM remote_path_mappings
ORDER BY host, remote_path;

-- name: UpdateRemotePathMapping :exec
UPDATE remote_path_mappings
SET host = ?, remote_path = ?, local_path = ?
WHERE id = ?;

-- name: DeleteRemotePathMapping :exec
DELETE FROM remote_path_mappings WHERE id = ?;
