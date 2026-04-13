package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
)

type sqliteSessionStore struct {
	pool *db.SQLitePool
}

// NewSQLiteSessionStore returns a SessionStore backed by a SQLite pool.
func NewSQLiteSessionStore(pool *db.SQLitePool) SessionStore {
	return &sqliteSessionStore{pool: pool}
}

func (s *sqliteSessionStore) Create(ctx context.Context, session Session) error {
	return s.pool.Write(ctx, func(ex db.Executor) error {
		_, err := ex.ExecContext(ctx,
			"INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
			session.Token, session.UserID, session.ExpiresAt.Format("2006-01-02 15:04:05"))
		return err
	})
}

func (s *sqliteSessionStore) GetByToken(ctx context.Context, token string) (Session, error) {
	var session Session
	err := s.pool.Read(ctx, func(q db.Querier) error {
		row := q.QueryRowContext(ctx,
			"SELECT token, user_id, expires_at, created_at FROM sessions WHERE token = ?",
			token)
		var expiresStr, createdStr string
		if err := row.Scan(&session.Token, &session.UserID, &expiresStr, &createdStr); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrSessionExpired
			}
			return err
		}
		session.ExpiresAt, _ = time.Parse("2006-01-02 15:04:05", expiresStr)
		session.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdStr)
		return nil
	})
	return session, err
}

func (s *sqliteSessionStore) DeleteByToken(ctx context.Context, token string) error {
	return s.pool.Write(ctx, func(ex db.Executor) error {
		_, err := ex.ExecContext(ctx, "DELETE FROM sessions WHERE token = ?", token)
		return err
	})
}

func (s *sqliteSessionStore) DeleteExpired(ctx context.Context) error {
	return s.pool.Write(ctx, func(ex db.Executor) error {
		_, err := ex.ExecContext(ctx,
			"DELETE FROM sessions WHERE expires_at < ?",
			time.Now().Format("2006-01-02 15:04:05"))
		return err
	})
}
