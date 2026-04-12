-- name: CreateIndexer :one
INSERT INTO indexers (name, implementation, settings, enable_rss, enable_automatic_search, enable_interactive_search, priority)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id, name, implementation, settings, enable_rss, enable_automatic_search, enable_interactive_search, priority, added;

-- name: GetIndexerByID :one
SELECT id, name, implementation, settings, enable_rss, enable_automatic_search, enable_interactive_search, priority, added
FROM indexers
WHERE id = ?;

-- name: ListIndexers :many
SELECT id, name, implementation, settings, enable_rss, enable_automatic_search, enable_interactive_search, priority, added
FROM indexers
ORDER BY name;

-- name: UpdateIndexer :exec
UPDATE indexers
SET name                      = ?,
    implementation            = ?,
    settings                  = ?,
    enable_rss                = ?,
    enable_automatic_search   = ?,
    enable_interactive_search = ?,
    priority                  = ?
WHERE id = ?;

-- name: DeleteIndexer :exec
DELETE FROM indexers WHERE id = ?;
