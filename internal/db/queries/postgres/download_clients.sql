-- name: CreateDownloadClient :one
INSERT INTO download_clients (name, implementation, settings, enable, priority, remove_completed_downloads, remove_failed_downloads)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, name, implementation, settings, enable, priority, remove_completed_downloads, remove_failed_downloads, added;

-- name: GetDownloadClientByID :one
SELECT id, name, implementation, settings, enable, priority, remove_completed_downloads, remove_failed_downloads, added
FROM download_clients
WHERE id = $1;

-- name: ListDownloadClients :many
SELECT id, name, implementation, settings, enable, priority, remove_completed_downloads, remove_failed_downloads, added
FROM download_clients
ORDER BY name;

-- name: UpdateDownloadClient :exec
UPDATE download_clients
SET name                       = $2,
    implementation             = $3,
    settings                   = $4,
    enable                     = $5,
    priority                   = $6,
    remove_completed_downloads = $7,
    remove_failed_downloads    = $8
WHERE id = $1;

-- name: DeleteDownloadClient :exec
DELETE FROM download_clients WHERE id = $1;
