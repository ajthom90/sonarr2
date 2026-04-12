-- name: CreateIndexer :one
INSERT INTO indexers (name, implementation, settings, enable_rss, enable_automatic_search, enable_interactive_search, priority)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, name, implementation, settings, enable_rss, enable_automatic_search, enable_interactive_search, priority, added;

-- name: GetIndexerByID :one
SELECT id, name, implementation, settings, enable_rss, enable_automatic_search, enable_interactive_search, priority, added
FROM indexers
WHERE id = $1;

-- name: ListIndexers :many
SELECT id, name, implementation, settings, enable_rss, enable_automatic_search, enable_interactive_search, priority, added
FROM indexers
ORDER BY name;

-- name: UpdateIndexer :exec
UPDATE indexers
SET name                      = $2,
    implementation            = $3,
    settings                  = $4,
    enable_rss                = $5,
    enable_automatic_search   = $6,
    enable_interactive_search = $7,
    priority                  = $8
WHERE id = $1;

-- name: DeleteIndexer :exec
DELETE FROM indexers WHERE id = $1;
