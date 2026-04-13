-- +goose Up
ALTER TABLE host_config ADD COLUMN tvdb_api_key TEXT NOT NULL DEFAULT '';

-- +goose Down
-- SQLite doesn't support DROP COLUMN easily; this is a best-effort rollback.
