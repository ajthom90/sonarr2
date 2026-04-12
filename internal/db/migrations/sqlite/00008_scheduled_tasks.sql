-- +goose Up
CREATE TABLE scheduled_tasks (
    type_name      TEXT PRIMARY KEY,
    interval_secs  INTEGER NOT NULL,
    last_execution TEXT,
    next_execution TEXT NOT NULL
);

-- +goose Down
DROP TABLE scheduled_tasks;
