-- +goose Up
CREATE TABLE commands (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    body        TEXT NOT NULL DEFAULT '{}',
    priority    INTEGER NOT NULL DEFAULT 2,
    status      TEXT NOT NULL DEFAULT 'queued',
    queued_at   TEXT NOT NULL DEFAULT (datetime('now')),
    started_at  TEXT,
    ended_at    TEXT,
    duration_ms INTEGER,
    exception   TEXT NOT NULL DEFAULT '',
    trigger     TEXT NOT NULL DEFAULT 'manual',
    message     TEXT NOT NULL DEFAULT '',
    result      TEXT NOT NULL DEFAULT '{}',
    worker_id   TEXT NOT NULL DEFAULT '',
    lease_until TEXT,
    dedup_key   TEXT NOT NULL DEFAULT ''
);

CREATE INDEX commands_claim_idx ON commands (priority ASC, queued_at ASC)
    WHERE status = 'queued';
CREATE INDEX commands_lease_sweep_idx ON commands (lease_until)
    WHERE status = 'running';
CREATE INDEX commands_dedup_idx ON commands (dedup_key)
    WHERE dedup_key != '' AND status IN ('queued', 'running');

-- +goose Down
DROP TABLE commands;
