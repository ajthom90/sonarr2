package scheduler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	pggen "github.com/ajthom90/sonarr2/internal/db/gen/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// postgresTaskStore implements TaskStore against a Postgres pool.
type postgresTaskStore struct {
	q *pggen.Queries
}

// NewPostgresTaskStore returns a TaskStore backed by pool.
func NewPostgresTaskStore(pool *db.PostgresPool) TaskStore {
	return &postgresTaskStore{q: pggen.New(pool.Raw())}
}

func (p *postgresTaskStore) GetDueTasks(ctx context.Context) ([]ScheduledTask, error) {
	rows, err := p.q.GetDueTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("scheduler: get due tasks: %w", err)
	}
	tasks := make([]ScheduledTask, 0, len(rows))
	for _, r := range rows {
		tasks = append(tasks, scheduledTaskFromPostgres(r))
	}
	return tasks, nil
}

func (p *postgresTaskStore) UpdateExecution(ctx context.Context, typeName string, nextExecution time.Time) error {
	err := p.q.UpdateTaskExecution(ctx, pggen.UpdateTaskExecutionParams{
		TypeName:      typeName,
		NextExecution: pgtype.Timestamptz{Time: nextExecution, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("scheduler: update execution: %w", err)
	}
	return nil
}

func (p *postgresTaskStore) Upsert(ctx context.Context, task ScheduledTask) error {
	err := p.q.UpsertScheduledTask(ctx, pggen.UpsertScheduledTaskParams{
		TypeName:      task.TypeName,
		IntervalSecs:  int32(task.IntervalSecs),
		NextExecution: pgtype.Timestamptz{Time: task.NextExecution, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("scheduler: upsert: %w", err)
	}
	return nil
}

func (p *postgresTaskStore) Get(ctx context.Context, typeName string) (ScheduledTask, error) {
	row, err := p.q.GetScheduledTask(ctx, typeName)
	if errors.Is(err, pgx.ErrNoRows) {
		return ScheduledTask{}, fmt.Errorf("scheduler: get %q: %w", typeName, ErrNotFound)
	}
	if err != nil {
		return ScheduledTask{}, fmt.Errorf("scheduler: get: %w", err)
	}
	return scheduledTaskFromPostgres(row), nil
}

func (p *postgresTaskStore) List(ctx context.Context) ([]ScheduledTask, error) {
	rows, err := p.q.ListScheduledTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("scheduler: list: %w", err)
	}
	tasks := make([]ScheduledTask, 0, len(rows))
	for _, r := range rows {
		tasks = append(tasks, scheduledTaskFromPostgres(r))
	}
	return tasks, nil
}

// scheduledTaskFromPostgres converts a sqlc-generated pggen.ScheduledTask row
// to the canonical scheduler.ScheduledTask struct.
func scheduledTaskFromPostgres(r pggen.ScheduledTask) ScheduledTask {
	return ScheduledTask{
		TypeName:      r.TypeName,
		IntervalSecs:  int(r.IntervalSecs),
		LastExecution: ptrFromPgTimestamptz(r.LastExecution),
		NextExecution: r.NextExecution.Time,
	}
}

// ptrFromPgTimestamptz converts a pgtype.Timestamptz to a *time.Time.
func ptrFromPgTimestamptz(v pgtype.Timestamptz) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}
