-- name: GetHostConfig :one
SELECT id, api_key, auth_mode, migration_state, created_at, updated_at
FROM host_config
WHERE id = 1;

-- name: UpsertHostConfig :exec
INSERT INTO host_config (id, api_key, auth_mode, migration_state)
VALUES (1, $1, $2, $3)
ON CONFLICT (id) DO UPDATE
SET api_key = EXCLUDED.api_key,
    auth_mode = EXCLUDED.auth_mode,
    migration_state = EXCLUDED.migration_state,
    updated_at = now();
