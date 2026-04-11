package db

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestOpenSQLiteInMemory(t *testing.T) {
	pool, err := OpenSQLite(context.Background(), SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if pool.Dialect() != DialectSQLite {
		t.Errorf("Dialect() = %q, want sqlite", pool.Dialect())
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

func TestOpenSQLiteFileDSN(t *testing.T) {
	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, "test.db") + "?_journal=WAL&_busy_timeout=5000"

	pool, err := OpenSQLite(context.Background(), SQLiteOptions{
		DSN:         dsn,
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if err := pool.Ping(context.Background()); err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

func TestSQLiteWriterSerializesWrites(t *testing.T) {
	pool, err := OpenSQLite(context.Background(), SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	// Create a test table via the writer.
	err = pool.Write(context.Background(), func(exec Executor) error {
		_, err := exec.ExecContext(context.Background(),
			`CREATE TABLE counter (n INTEGER)`)
		if err != nil {
			return err
		}
		_, err = exec.ExecContext(context.Background(),
			`INSERT INTO counter (n) VALUES (0)`)
		return err
	})
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Launch 20 concurrent writers. Single-writer discipline means they
	// all complete without database-is-locked errors.
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := pool.Write(context.Background(), func(exec Executor) error {
				_, err := exec.ExecContext(context.Background(),
					`UPDATE counter SET n = n + 1`)
				return err
			})
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent write error: %v", err)
	}

	// Verify the final count is 20.
	var n int
	err = pool.Read(context.Background(), func(q Querier) error {
		row := q.QueryRowContext(context.Background(), `SELECT n FROM counter`)
		return row.Scan(&n)
	})
	if err != nil {
		t.Fatalf("read counter: %v", err)
	}
	if n != 20 {
		t.Errorf("counter = %d, want 20", n)
	}
}

func TestOpenSQLiteRejectsEmptyDSN(t *testing.T) {
	_, err := OpenSQLite(context.Background(), SQLiteOptions{DSN: ""})
	if err == nil {
		t.Fatal("expected error for empty DSN")
	}
}
