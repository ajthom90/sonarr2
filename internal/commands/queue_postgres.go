package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	pggen "github.com/ajthom90/sonarr2/internal/db/gen/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// postgresQueue implements Queue against a Postgres pool.
type postgresQueue struct {
	q *pggen.Queries
}

// NewPostgresQueue returns a Queue backed by pool.
func NewPostgresQueue(pool *db.PostgresPool) Queue {
	return &postgresQueue{q: pggen.New(pool.Raw())}
}

func (p *postgresQueue) Enqueue(ctx context.Context, name string, body json.RawMessage, priority Priority, trigger Trigger, dedupKey string) (Command, error) {
	row, err := p.q.EnqueueCommand(ctx, pggen.EnqueueCommandParams{
		Name:     name,
		Body:     []byte(body),
		Priority: int16(priority),
		Trigger:  string(trigger),
		DedupKey: dedupKey,
	})
	if err != nil {
		return Command{}, fmt.Errorf("commands: enqueue: %w", err)
	}
	return commandFromPostgres(row), nil
}

func (p *postgresQueue) Claim(ctx context.Context, workerID string, leaseDuration time.Duration) (*Command, error) {
	row, err := p.q.ClaimCommand(ctx, pggen.ClaimCommandParams{
		WorkerID:   workerID,
		LeaseUntil: pgtype.Timestamptz{Time: time.Now().Add(leaseDuration), Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("commands: claim: %w", err)
	}
	cmd := commandFromPostgres(row)
	return &cmd, nil
}

func (p *postgresQueue) Complete(ctx context.Context, id int64, durationMs int64, result json.RawMessage, message string) error {
	err := p.q.CompleteCommand(ctx, pggen.CompleteCommandParams{
		ID:         id,
		DurationMs: pgtype.Int8{Int64: durationMs, Valid: true},
		Result:     []byte(result),
		Message:    message,
	})
	if err != nil {
		return fmt.Errorf("commands: complete: %w", err)
	}
	return nil
}

func (p *postgresQueue) Fail(ctx context.Context, id int64, durationMs int64, exception string, message string) error {
	err := p.q.FailCommand(ctx, pggen.FailCommandParams{
		ID:         id,
		DurationMs: pgtype.Int8{Int64: durationMs, Valid: true},
		Exception:  exception,
		Message:    message,
	})
	if err != nil {
		return fmt.Errorf("commands: fail: %w", err)
	}
	return nil
}

func (p *postgresQueue) RefreshLease(ctx context.Context, id int64, leaseDuration time.Duration) error {
	err := p.q.RefreshLease(ctx, pggen.RefreshLeaseParams{
		ID:         id,
		LeaseUntil: pgtype.Timestamptz{Time: time.Now().Add(leaseDuration), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("commands: refresh lease: %w", err)
	}
	return nil
}

func (p *postgresQueue) SweepExpiredLeases(ctx context.Context) (int64, error) {
	n, err := p.q.SweepExpiredLeases(ctx)
	if err != nil {
		return 0, fmt.Errorf("commands: sweep expired leases: %w", err)
	}
	return n, nil
}

func (p *postgresQueue) FindDuplicate(ctx context.Context, dedupKey string) (int64, bool, error) {
	id, err := p.q.FindDuplicate(ctx, dedupKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("commands: find duplicate: %w", err)
	}
	return id, true, nil
}

func (p *postgresQueue) DeleteOldCompleted(ctx context.Context, olderThan time.Time) (int64, error) {
	n, err := p.q.DeleteOldCompleted(ctx, pgtype.Timestamptz{Time: olderThan, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("commands: delete old completed: %w", err)
	}
	return n, nil
}

func (p *postgresQueue) Get(ctx context.Context, id int64) (Command, error) {
	row, err := p.q.GetCommand(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Command{}, fmt.Errorf("commands: get %d: %w", id, ErrNotFound)
	}
	if err != nil {
		return Command{}, fmt.Errorf("commands: get: %w", err)
	}
	return commandFromPostgres(row), nil
}

// commandFromPostgres converts a sqlc-generated pggen.Command row to the
// canonical commands.Command struct.
func commandFromPostgres(r pggen.Command) Command {
	return Command{
		ID:         r.ID,
		Name:       r.Name,
		Body:       json.RawMessage(r.Body),
		Priority:   Priority(r.Priority),
		Status:     Status(r.Status),
		QueuedAt:   r.QueuedAt.Time,
		StartedAt:  ptrFromPgTimestamptz(r.StartedAt),
		EndedAt:    ptrFromPgTimestamptz(r.EndedAt),
		DurationMs: ptrFromPgInt8(r.DurationMs),
		Exception:  r.Exception,
		Trigger:    Trigger(r.Trigger),
		Message:    r.Message,
		Result:     json.RawMessage(r.Result),
		WorkerID:   r.WorkerID,
		LeaseUntil: ptrFromPgTimestamptz(r.LeaseUntil),
		DedupKey:   r.DedupKey,
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

// ptrFromPgInt8 converts a pgtype.Int8 to a *int64.
func ptrFromPgInt8(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	n := v.Int64
	return &n
}
