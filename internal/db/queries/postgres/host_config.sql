-- name: GetHostConfig :one
SELECT id, api_key, auth_mode, migration_state, tvdb_api_key,
       recycle_bin, recycle_bin_cleanup_days, created_at, updated_at
FROM host_config
WHERE id = 1;

-- name: UpsertHostConfig :exec
INSERT INTO host_config (id, api_key, auth_mode, migration_state, tvdb_api_key,
                         recycle_bin, recycle_bin_cleanup_days)
VALUES (1, $1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO UPDATE
SET api_key = EXCLUDED.api_key,
    auth_mode = EXCLUDED.auth_mode,
    migration_state = EXCLUDED.migration_state,
    tvdb_api_key = EXCLUDED.tvdb_api_key,
    recycle_bin = EXCLUDED.recycle_bin,
    recycle_bin_cleanup_days = EXCLUDED.recycle_bin_cleanup_days,
    updated_at = now();
