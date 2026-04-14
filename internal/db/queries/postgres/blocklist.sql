-- name: CreateBlocklist :one
INSERT INTO blocklist (
    series_id, episode_ids, source_title, quality, languages,
    date, published_date, size, protocol, indexer, indexer_flags,
    release_type, message, torrent_info_hash
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING id, series_id, episode_ids, source_title, quality, languages,
          date, published_date, size, protocol, indexer, indexer_flags,
          release_type, message, torrent_info_hash;

-- name: GetBlocklistByID :one
SELECT id, series_id, episode_ids, source_title, quality, languages,
       date, published_date, size, protocol, indexer, indexer_flags,
       release_type, message, torrent_info_hash
FROM blocklist WHERE id = $1;

-- name: ListBlocklist :many
SELECT id, series_id, episode_ids, source_title, quality, languages,
       date, published_date, size, protocol, indexer, indexer_flags,
       release_type, message, torrent_info_hash
FROM blocklist
ORDER BY date DESC
LIMIT $1 OFFSET $2;

-- name: CountBlocklist :one
SELECT COUNT(*) FROM blocklist;

-- name: ListBlocklistBySeries :many
SELECT id, series_id, episode_ids, source_title, quality, languages,
       date, published_date, size, protocol, indexer, indexer_flags,
       release_type, message, torrent_info_hash
FROM blocklist
WHERE series_id = $1
ORDER BY date DESC;

-- name: DeleteBlocklist :exec
DELETE FROM blocklist WHERE id = $1;

-- name: DeleteBlocklistBySeries :exec
DELETE FROM blocklist WHERE series_id = $1;

-- name: ClearBlocklist :exec
DELETE FROM blocklist;
