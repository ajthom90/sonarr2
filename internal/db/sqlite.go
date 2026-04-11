package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteOptions controls the SQLite connection pool.
type SQLiteOptions struct {
	DSN         string
	BusyTimeout time.Duration
}

// Executor is the interface satisfied by both *sql.DB and *sql.Tx for exec
// statements. Used by Write callbacks.
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// Querier is the interface satisfied by a read-only *sql.DB for queries.
// Used by Read callbacks.
type Querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// SQLitePool owns two *sql.DB handles: a single-connection writer DB and a
// multi-connection read-only DB. All writes are serialized through a
// writer goroutine, which eliminates "database is locked" errors at the
// application layer.
type SQLitePool struct {
	writer    *sql.DB // max open conns = 1; PRAGMA journal_mode=WAL
	reader    *sql.DB // query_only; multi-conn
	writeReq  chan writeReq
	wg        sync.WaitGroup
	closeOnce sync.Once
	closed    chan struct{}
}

type writeReq struct {
	ctx  context.Context
	fn   func(Executor) error
	done chan error
}

// Dialect implements Pool.
func (p *SQLitePool) Dialect() Dialect { return DialectSQLite }

// Ping implements Pool.
func (p *SQLitePool) Ping(ctx context.Context) error {
	if err := p.reader.PingContext(ctx); err != nil {
		return err
	}
	return p.writer.PingContext(ctx)
}

// Close implements Pool. Shuts down the writer loop and closes both sql.DB
// handles.
func (p *SQLitePool) Close() error {
	var err error
	p.closeOnce.Do(func() {
		close(p.closed)
		p.wg.Wait()
		if rerr := p.reader.Close(); rerr != nil {
			err = rerr
		}
		if werr := p.writer.Close(); werr != nil && err == nil {
			err = werr
		}
	})
	return err
}

// Write submits fn to the writer goroutine and blocks until it completes.
// fn receives an Executor that wraps the single writer connection; writes
// are serialized with respect to all other Write calls.
func (p *SQLitePool) Write(ctx context.Context, fn func(Executor) error) error {
	req := writeReq{ctx: ctx, fn: fn, done: make(chan error, 1)}
	select {
	case p.writeReq <- req:
	case <-p.closed:
		return errors.New("db: sqlite pool closed")
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-req.done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Read executes fn against the read-only connection pool. Multiple Read
// calls may execute concurrently.
func (p *SQLitePool) Read(ctx context.Context, fn func(Querier) error) error {
	return fn(p.reader)
}

// RawWriter returns the underlying writer *sql.DB for use by sqlc-generated
// code that needs a DBTX. Only used by Store implementations.
func (p *SQLitePool) RawWriter() *sql.DB { return p.writer }

// RawReader returns the underlying reader *sql.DB for read-only query
// execution by sqlc-generated code.
func (p *SQLitePool) RawReader() *sql.DB { return p.reader }

// normalizeDSN rewrites the bare ":memory:" DSN to a URI that enables shared
// cache so that the reader and writer sql.DB handles share the same in-memory
// database. File-path and URI DSNs are returned unchanged.
func normalizeDSN(dsn string) string {
	if dsn == ":memory:" {
		return "file::memory:?cache=shared&mode=memory"
	}
	return dsn
}

// OpenSQLite opens a SQLite database using the modernc.org/sqlite pure-Go
// driver. It creates two sql.DB handles: a single-connection writer and a
// multi-connection reader with PRAGMA query_only=1.
func OpenSQLite(ctx context.Context, opts SQLiteOptions) (*SQLitePool, error) {
	if opts.DSN == "" {
		return nil, errors.New("db: sqlite DSN is required")
	}

	dsn := normalizeDSN(opts.DSN)

	writer, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open sqlite writer: %w", err)
	}
	writer.SetMaxOpenConns(1)
	writer.SetMaxIdleConns(1)

	if err := writer.PingContext(ctx); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("db: sqlite writer ping: %w", err)
	}

	// Apply WAL mode and busy_timeout via PRAGMA for safety.
	if _, err := writer.ExecContext(ctx, `PRAGMA journal_mode=WAL`); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("db: set WAL mode: %w", err)
	}
	if opts.BusyTimeout > 0 {
		ms := int(opts.BusyTimeout / time.Millisecond)
		if _, err := writer.ExecContext(ctx, fmt.Sprintf("PRAGMA busy_timeout=%d", ms)); err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("db: set busy_timeout: %w", err)
		}
	}

	reader, err := sql.Open("sqlite", dsn)
	if err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("db: open sqlite reader: %w", err)
	}
	if _, err := reader.ExecContext(ctx, `PRAGMA query_only=1`); err != nil {
		_ = reader.Close()
		_ = writer.Close()
		return nil, fmt.Errorf("db: set query_only: %w", err)
	}

	p := &SQLitePool{
		writer:   writer,
		reader:   reader,
		writeReq: make(chan writeReq),
		closed:   make(chan struct{}),
	}

	p.wg.Add(1)
	go p.writerLoop()

	return p, nil
}

// writerLoop is the single goroutine that owns the writer connection.
// Every Write call flows through here, guaranteeing serial access.
func (p *SQLitePool) writerLoop() {
	defer p.wg.Done()
	for {
		select {
		case <-p.closed:
			return
		case req := <-p.writeReq:
			err := req.fn(p.writer)
			req.done <- err
		}
	}
}
