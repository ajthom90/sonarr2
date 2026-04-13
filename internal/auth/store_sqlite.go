package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
)

type sqliteUserStore struct {
	pool *db.SQLitePool
}

// NewSQLiteUserStore returns a UserStore backed by a SQLite pool.
func NewSQLiteUserStore(pool *db.SQLitePool) UserStore {
	return &sqliteUserStore{pool: pool}
}

func (s *sqliteUserStore) Create(ctx context.Context, username, passwordHash string) (User, error) {
	var user User
	err := s.pool.Write(ctx, func(ex db.Executor) error {
		result, err := ex.ExecContext(ctx,
			"INSERT INTO users (username, password_hash) VALUES (?, ?)",
			username, passwordHash)
		if err != nil {
			return err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		user = User{
			ID:           id,
			Username:     username,
			PasswordHash: passwordHash,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		return nil
	})
	return user, err
}

func (s *sqliteUserStore) GetByUsername(ctx context.Context, username string) (User, error) {
	var user User
	err := s.pool.Read(ctx, func(q db.Querier) error {
		row := q.QueryRowContext(ctx,
			"SELECT id, username, password_hash, created_at, updated_at FROM users WHERE username = ?",
			username)
		var createdStr, updatedStr string
		if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &createdStr, &updatedStr); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrUserNotFound
			}
			return err
		}
		user.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdStr)
		user.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedStr)
		return nil
	})
	return user, err
}

func (s *sqliteUserStore) Count(ctx context.Context) (int, error) {
	var count int
	err := s.pool.Read(ctx, func(q db.Querier) error {
		return q.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	})
	return count, err
}
