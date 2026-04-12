-- name: GetDueTasks :many
SELECT * FROM scheduled_tasks
WHERE next_execution <= now()
ORDER BY next_execution ASC;

-- name: UpdateTaskExecution :exec
UPDATE scheduled_tasks
SET last_execution = now(),
    next_execution = $2
WHERE type_name = $1;

-- name: UpsertScheduledTask :exec
INSERT INTO scheduled_tasks (type_name, interval_secs, next_execution)
VALUES ($1, $2, $3)
ON CONFLICT (type_name) DO UPDATE
SET interval_secs = EXCLUDED.interval_secs;

-- name: GetScheduledTask :one
SELECT * FROM scheduled_tasks WHERE type_name = $1;

-- name: ListScheduledTasks :many
SELECT * FROM scheduled_tasks ORDER BY type_name;
