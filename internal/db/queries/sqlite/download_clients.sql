-- name: CreateDownloadClient :one
INSERT INTO download_clients (name, implementation, settings, enable, priority, remove_completed_downloads, remove_failed_downloads)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id, name, implementation, settings, enable, priority, remove_completed_downloads, remove_failed_downloads, added;

-- name: GetDownloadClientByID :one
SELECT id, name, implementation, settings, enable, priority, remove_completed_downloads, remove_failed_downloads, added
FROM download_clients
WHERE id = ?;

-- name: ListDownloadClients :many
SELECT id, name, implementation, settings, enable, priority, remove_completed_downloads, remove_failed_downloads, added
FROM download_clients
ORDER BY name;

-- name: UpdateDownloadClient :exec
UPDATE download_clients
SET name                       = ?,
    implementation             = ?,
    settings                   = ?,
    enable                     = ?,
    priority                   = ?,
    remove_completed_downloads = ?,
    remove_failed_downloads    = ?
WHERE id = ?;

-- name: DeleteDownloadClient :exec
DELETE FROM download_clients WHERE id = ?;
