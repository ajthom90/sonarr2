-- name: CreateNotification :one
INSERT INTO notifications (name, implementation, settings, on_grab, on_download, on_health_issue, tags)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id, name, implementation, settings, on_grab, on_download, on_health_issue, tags, added;

-- name: GetNotificationByID :one
SELECT id, name, implementation, settings, on_grab, on_download, on_health_issue, tags, added
FROM notifications
WHERE id = ?;

-- name: ListNotifications :many
SELECT id, name, implementation, settings, on_grab, on_download, on_health_issue, tags, added
FROM notifications
ORDER BY name;

-- name: UpdateNotification :exec
UPDATE notifications
SET name            = ?,
    implementation  = ?,
    settings        = ?,
    on_grab         = ?,
    on_download     = ?,
    on_health_issue = ?,
    tags            = ?
WHERE id = ?;

-- name: DeleteNotification :exec
DELETE FROM notifications WHERE id = ?;
