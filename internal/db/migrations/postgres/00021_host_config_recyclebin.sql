-- SPDX-License-Identifier: GPL-3.0-or-later
-- Ported from Sonarr's RecycleBin/RecycleBinCleanupDays config
-- (src/NzbDrone.Core/Configuration/ConfigService.cs).
-- Copyright (c) Team Sonarr, licensed under GPL-3.0.

-- +goose Up
ALTER TABLE host_config ADD COLUMN recycle_bin TEXT NOT NULL DEFAULT '';
ALTER TABLE host_config ADD COLUMN recycle_bin_cleanup_days INTEGER NOT NULL DEFAULT 7;

-- +goose Down
ALTER TABLE host_config DROP COLUMN recycle_bin_cleanup_days;
ALTER TABLE host_config DROP COLUMN recycle_bin;
