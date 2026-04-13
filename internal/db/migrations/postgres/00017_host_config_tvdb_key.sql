-- +goose Up
ALTER TABLE host_config ADD COLUMN tvdb_api_key TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE host_config DROP COLUMN tvdb_api_key;
