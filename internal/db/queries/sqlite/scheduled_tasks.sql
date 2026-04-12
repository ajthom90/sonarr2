-- name: GetDueTasks :many
SELECT * FROM scheduled_tasks
WHERE next_execution <= datetime('now')
ORDER BY next_execution ASC;

-- name: UpdateTaskExecution :exec
UPDATE scheduled_tasks
SET last_execution = datetime('now'),
    next_execution = ?
WHERE type_name = ?;

-- name: UpsertScheduledTask :exec
INSERT INTO scheduled_tasks (type_name, interval_secs, next_execution)
VALUES (?, ?, ?)
ON CONFLICT (type_name) DO UPDATE
SET interval_secs = excluded.interval_secs;

-- name: GetScheduledTask :one
SELECT * FROM scheduled_tasks WHERE type_name = ?;

-- name: ListScheduledTasks :many
SELECT * FROM scheduled_tasks ORDER BY type_name;
