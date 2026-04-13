-- +goose Up
CREATE TABLE notifications (
    id              SERIAL PRIMARY KEY,
    name            TEXT NOT NULL,
    implementation  TEXT NOT NULL,
    settings        JSONB NOT NULL DEFAULT '{}',
    on_grab         BOOLEAN NOT NULL DEFAULT TRUE,
    on_download     BOOLEAN NOT NULL DEFAULT TRUE,
    on_health_issue BOOLEAN NOT NULL DEFAULT TRUE,
    tags            JSONB NOT NULL DEFAULT '[]',
    added           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE notifications;
