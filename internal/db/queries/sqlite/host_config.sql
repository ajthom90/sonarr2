-- name: GetHostConfig :one
SELECT id, api_key, auth_mode, migration_state, tvdb_api_key, created_at, updated_at
FROM host_config
WHERE id = 1;

-- name: UpsertHostConfig :exec
INSERT INTO host_config (id, api_key, auth_mode, migration_state, tvdb_api_key)
VALUES (1, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE
SET api_key = excluded.api_key,
    auth_mode = excluded.auth_mode,
    migration_state = excluded.migration_state,
    tvdb_api_key = excluded.tvdb_api_key,
    updated_at = datetime('now');
