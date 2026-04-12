-- name: EnqueueCommand :one
INSERT INTO commands (name, body, priority, trigger, dedup_key)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ClaimCommand :one
UPDATE commands
SET status = 'running',
    worker_id = $1,
    started_at = now(),
    lease_until = $2
WHERE id = (
    SELECT id FROM commands
    WHERE status = 'queued'
    ORDER BY priority ASC, queued_at ASC
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING *;

-- name: CompleteCommand :exec
UPDATE commands
SET status = 'completed',
    ended_at = now(),
    duration_ms = $2,
    result = $3,
    message = $4
WHERE id = $1;

-- name: FailCommand :exec
UPDATE commands
SET status = 'failed',
    ended_at = now(),
    duration_ms = $2,
    exception = $3,
    message = $4
WHERE id = $1;

-- name: GetCommand :one
SELECT * FROM commands WHERE id = $1;

-- name: RefreshLease :exec
UPDATE commands SET lease_until = $2 WHERE id = $1 AND status = 'running';

-- name: SweepExpiredLeases :execrows
UPDATE commands
SET status = 'queued', worker_id = '', started_at = NULL, lease_until = NULL
WHERE status = 'running' AND lease_until < now();

-- name: FindDuplicate :one
SELECT id FROM commands
WHERE dedup_key = $1 AND status IN ('queued', 'running')
LIMIT 1;

-- name: DeleteOldCompleted :execrows
DELETE FROM commands
WHERE status IN ('completed', 'failed') AND ended_at < $1;
