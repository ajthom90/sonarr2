package auth

import (
	"context"
	"errors"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/jackc/pgx/v5"
)

type postgresUserStore struct {
	pool *db.PostgresPool
}

// NewPostgresUserStore returns a UserStore backed by a Postgres pool.
func NewPostgresUserStore(pool *db.PostgresPool) UserStore {
	return &postgresUserStore{pool: pool}
}

func (s *postgresUserStore) Create(ctx context.Context, username, passwordHash string) (User, error) {
	var user User
	err := s.pool.Raw().QueryRow(ctx,
		"INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id, username, password_hash, created_at, updated_at",
		username, passwordHash,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *postgresUserStore) GetByUsername(ctx context.Context, username string) (User, error) {
	var user User
	err := s.pool.Raw().QueryRow(ctx,
		"SELECT id, username, password_hash, created_at, updated_at FROM users WHERE username = $1",
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *postgresUserStore) Count(ctx context.Context) (int, error) {
	var count int
	err := s.pool.Raw().QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

type postgresSessionStore struct {
	pool *db.PostgresPool
}

// NewPostgresSessionStore returns a SessionStore backed by a Postgres pool.
func NewPostgresSessionStore(pool *db.PostgresPool) SessionStore {
	return &postgresSessionStore{pool: pool}
}

func (s *postgresSessionStore) Create(ctx context.Context, session Session) error {
	_, err := s.pool.Raw().Exec(ctx,
		"INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)",
		session.Token, session.UserID, session.ExpiresAt,
	)
	return err
}

func (s *postgresSessionStore) GetByToken(ctx context.Context, token string) (Session, error) {
	var session Session
	err := s.pool.Raw().QueryRow(ctx,
		"SELECT token, user_id, expires_at, created_at FROM sessions WHERE token = $1",
		token,
	).Scan(&session.Token, &session.UserID, &session.ExpiresAt, &session.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrSessionExpired
	}
	if err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *postgresSessionStore) DeleteByToken(ctx context.Context, token string) error {
	_, err := s.pool.Raw().Exec(ctx,
		"DELETE FROM sessions WHERE token = $1",
		token,
	)
	return err
}

func (s *postgresSessionStore) DeleteExpired(ctx context.Context) error {
	_, err := s.pool.Raw().Exec(ctx,
		"DELETE FROM sessions WHERE expires_at < now()",
	)
	return err
}
