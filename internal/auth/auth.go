// Package auth provides user authentication with bcrypt password hashing
// and session management for sonarr2.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	ErrUserNotFound       = errors.New("auth: user not found")
	ErrSessionExpired     = errors.New("auth: session expired")
	ErrUserExists         = errors.New("auth: user already exists")
)

// User represents an authenticated user.
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Session represents an active login session.
type Session struct {
	Token     string
	UserID    int64
	ExpiresAt time.Time
	CreatedAt time.Time
}

// UserStore provides CRUD for users.
type UserStore interface {
	Create(ctx context.Context, username, passwordHash string) (User, error)
	GetByUsername(ctx context.Context, username string) (User, error)
	Count(ctx context.Context) (int, error)
}

// SessionStore provides CRUD for sessions.
type SessionStore interface {
	Create(ctx context.Context, session Session) error
	GetByToken(ctx context.Context, token string) (Session, error)
	DeleteByToken(ctx context.Context, token string) error
	DeleteExpired(ctx context.Context) error
}

// HashPassword hashes a password using bcrypt.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword verifies a password against a bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// NewSessionToken generates a cryptographically random session token.
func NewSessionToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// SessionTTL is the default session lifetime.
const SessionTTL = 7 * 24 * time.Hour // 7 days
