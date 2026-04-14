-- name: CreateBlocklist :one
INSERT INTO blocklist (
    series_id, episode_ids, source_title, quality, languages,
    date, published_date, size, protocol, indexer, indexer_flags,
    release_type, message, torrent_info_hash
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, series_id, episode_ids, source_title, quality, languages,
          date, published_date, size, protocol, indexer, indexer_flags,
          release_type, message, torrent_info_hash;

-- name: GetBlocklistByID :one
SELECT id, series_id, episode_ids, source_title, quality, languages,
       date, published_date, size, protocol, indexer, indexer_flags,
       release_type, message, torrent_info_hash
FROM blocklist WHERE id = ?;

-- name: ListBlocklist :many
SELECT id, series_id, episode_ids, source_title, quality, languages,
       date, published_date, size, protocol, indexer, indexer_flags,
       release_type, message, torrent_info_hash
FROM blocklist
ORDER BY date DESC
LIMIT ? OFFSET ?;

-- name: CountBlocklist :one
SELECT COUNT(*) FROM blocklist;

-- name: ListBlocklistBySeries :many
SELECT id, series_id, episode_ids, source_title, quality, languages,
       date, published_date, size, protocol, indexer, indexer_flags,
       release_type, message, torrent_info_hash
FROM blocklist
WHERE series_id = ?
ORDER BY date DESC;

-- name: DeleteBlocklist :exec
DELETE FROM blocklist WHERE id = ?;

-- name: DeleteBlocklistBySeries :exec
DELETE FROM blocklist WHERE series_id = ?;

-- name: ClearBlocklist :exec
DELETE FROM blocklist;
