-- name: CreateRemotePathMapping :one
INSERT INTO remote_path_mappings (host, remote_path, local_path)
VALUES ($1, $2, $3)
RETURNING id, host, remote_path, local_path;

-- name: GetRemotePathMappingByID :one
SELECT id, host, remote_path, local_path
FROM remote_path_mappings WHERE id = $1;

-- name: ListRemotePathMappings :many
SELECT id, host, remote_path, local_path
FROM remote_path_mappings
ORDER BY host, remote_path;

-- name: UpdateRemotePathMapping :exec
UPDATE remote_path_mappings
SET host = $1, remote_path = $2, local_path = $3
WHERE id = $4;

-- name: DeleteRemotePathMapping :exec
DELETE FROM remote_path_mappings WHERE id = $1;
