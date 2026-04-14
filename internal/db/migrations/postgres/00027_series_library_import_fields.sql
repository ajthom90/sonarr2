-- +goose Up
-- Postgres 11+ stores column defaults in metadata; ALTER TABLE is fast and non-blocking.
ALTER TABLE series ADD COLUMN quality_profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL;
ALTER TABLE series ADD COLUMN season_folder BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE series ADD COLUMN monitor_new_items TEXT NOT NULL DEFAULT 'all';
CREATE INDEX series_quality_profile_id_idx ON series (quality_profile_id);
UPDATE series SET quality_profile_id = 1
WHERE quality_profile_id IS NULL
  AND EXISTS (SELECT 1 FROM quality_profiles WHERE id = 1);

-- +goose Down
DROP INDEX series_quality_profile_id_idx;
ALTER TABLE series DROP COLUMN monitor_new_items;
ALTER TABLE series DROP COLUMN season_folder;
ALTER TABLE series DROP COLUMN quality_profile_id;
