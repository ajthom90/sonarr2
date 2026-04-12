package db

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestSQLiteFilePath(t *testing.T) {
	cases := map[string]struct {
		dsn  string
		want string
	}{
		"empty":             {"", ""},
		"bare memory":       {":memory:", ""},
		"uri memory":        {"file::memory:", ""},
		"uri memory with q": {"file::memory:?cache=shared&mode=memory", ""},
		"bare relative":     {"./data/sonarr2.db", "./data/sonarr2.db"},
		"bare absolute":     {"/var/lib/sonarr2/sonarr2.db", "/var/lib/sonarr2/sonarr2.db"},
		"uri relative":      {"file:./data/sonarr2.db", "./data/sonarr2.db"},
		"uri relative w q":  {"file:./data/sonarr2.db?_journal=WAL&_busy_timeout=5000", "./data/sonarr2.db"},
		"uri absolute w q":  {"file:/var/lib/sonarr2/x.db?_journal=WAL", "/var/lib/sonarr2/x.db"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := sqliteFilePath(tc.dsn)
			if got != tc.want {
				t.Errorf("sqliteFilePath(%q) = %q, want %q", tc.dsn, got, tc.want)
			}
		})
	}
}

func TestEnsureSQLiteDirCreatesMissingParent(t *testing.T) {
	base := t.TempDir()
	nested := filepath.Join(base, "one", "two", "three")
	dsn := "file:" + filepath.Join(nested, "sonarr2.db") + "?_journal=WAL"

	if err := ensureSQLiteDir(dsn); err != nil {
		t.Fatalf("ensureSQLiteDir: %v", err)
	}
	if _, err := os.Stat(nested); err != nil {
		t.Errorf("parent dir not created: %v", err)
	}
}

func TestEnsureSQLiteDirMemoryIsNoop(t *testing.T) {
	for _, dsn := range []string{":memory:", "file::memory:?cache=shared&mode=memory", ""} {
		if err := ensureSQLiteDir(dsn); err != nil {
			t.Errorf("ensureSQLiteDir(%q) = %v, want nil", dsn, err)
		}
	}
}

func TestOpenSQLiteCreatesParentDirForFileDSN(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "fresh", "subdir", "sonarr2.db")
	dsn := "file:" + path + "?_journal=WAL&_busy_timeout=5000"

	pool, err := OpenSQLite(context.Background(), SQLiteOptions{
		DSN:         dsn,
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if _, err := os.Stat(path); err != nil {
		t.Errorf("db file not created at %q: %v", path, err)
	}
}

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
