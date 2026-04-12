-- name: EnqueueCommand :one
INSERT INTO commands (name, body, priority, trigger, dedup_key)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: SelectNextQueuedCommand :one
SELECT id FROM commands
WHERE status = 'queued'
ORDER BY priority ASC, queued_at ASC
LIMIT 1;

-- name: MarkCommandRunning :exec
UPDATE commands
SET status = 'running',
    worker_id = ?,
    started_at = datetime('now'),
    lease_until = ?
WHERE id = ?;

-- name: CompleteCommand :exec
UPDATE commands
SET status = 'completed',
    ended_at = datetime('now'),
    duration_ms = ?,
    result = ?,
    message = ?
WHERE id = ?;

-- name: FailCommand :exec
UPDATE commands
SET status = 'failed',
    ended_at = datetime('now'),
    duration_ms = ?,
    exception = ?,
    message = ?
WHERE id = ?;

-- name: GetCommand :one
SELECT * FROM commands WHERE id = ?;

-- name: RefreshLease :exec
UPDATE commands SET lease_until = ? WHERE id = ? AND status = 'running';

-- name: SweepExpiredLeases :execrows
UPDATE commands
SET status = 'queued', worker_id = '', started_at = NULL, lease_until = NULL
WHERE status = 'running' AND lease_until < datetime('now');

-- name: FindDuplicate :one
SELECT id FROM commands
WHERE dedup_key = ? AND status IN ('queued', 'running')
LIMIT 1;

-- name: DeleteOldCompleted :execrows
DELETE FROM commands
WHERE status IN ('completed', 'failed') AND ended_at < ?;
