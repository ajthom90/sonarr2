-- name: GetHostConfig :one
SELECT id, api_key, auth_mode, migration_state, tvdb_api_key,
       recycle_bin, recycle_bin_cleanup_days, created_at, updated_at
FROM host_config
WHERE id = 1;

-- name: UpsertHostConfig :exec
INSERT INTO host_config (id, api_key, auth_mode, migration_state, tvdb_api_key,
                         recycle_bin, recycle_bin_cleanup_days)
VALUES (1, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE
SET api_key = excluded.api_key,
    auth_mode = excluded.auth_mode,
    migration_state = excluded.migration_state,
    tvdb_api_key = excluded.tvdb_api_key,
    recycle_bin = excluded.recycle_bin,
    recycle_bin_cleanup_days = excluded.recycle_bin_cleanup_days,
    updated_at = datetime('now');
