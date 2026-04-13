-- name: CreateNotification :one
INSERT INTO notifications (name, implementation, settings, on_grab, on_download, on_health_issue, tags)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, name, implementation, settings, on_grab, on_download, on_health_issue, tags, added;

-- name: GetNotificationByID :one
SELECT id, name, implementation, settings, on_grab, on_download, on_health_issue, tags, added
FROM notifications
WHERE id = $1;

-- name: ListNotifications :many
SELECT id, name, implementation, settings, on_grab, on_download, on_health_issue, tags, added
FROM notifications
ORDER BY name;

-- name: UpdateNotification :exec
UPDATE notifications
SET name            = $2,
    implementation  = $3,
    settings        = $4,
    on_grab         = $5,
    on_download     = $6,
    on_health_issue = $7,
    tags            = $8
WHERE id = $1;

-- name: DeleteNotification :exec
DELETE FROM notifications WHERE id = $1;
