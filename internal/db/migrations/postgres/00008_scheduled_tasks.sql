-- +goose Up
CREATE TABLE scheduled_tasks (
    type_name      TEXT PRIMARY KEY,
    interval_secs  INTEGER NOT NULL,
    last_execution TIMESTAMPTZ,
    next_execution TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE scheduled_tasks;
