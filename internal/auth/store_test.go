package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/auth"
	"github.com/ajthom90/sonarr2/internal/db"
)

func newTestStores(t *testing.T) (auth.UserStore, auth.SessionStore) {
	t.Helper()
	ctx := context.Background()
	pool, err := db.OpenSQLite(ctx, db.SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return auth.NewSQLiteUserStore(pool), auth.NewSQLiteSessionStore(pool)
}

func TestUserStoreCreateAndGet(t *testing.T) {
	users, _ := newTestStores(t)
	ctx := context.Background()

	hash, err := auth.HashPassword("testpass")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	// Count should start at 0.
	count, err := users.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 0 {
		t.Errorf("initial count = %d, want 0", count)
	}

	// Create a user.
	user, err := users.Create(ctx, "admin", hash)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if user.ID == 0 {
		t.Error("user.ID must be non-zero")
	}
	if user.Username != "admin" {
		t.Errorf("Username = %q, want %q", user.Username, "admin")
	}

	// Count should be 1.
	count, err = users.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Errorf("count after create = %d, want 1", count)
	}

	// GetByUsername should find the user.
	got, err := users.GetByUsername(ctx, "admin")
	if err != nil {
		t.Fatalf("GetByUsername: %v", err)
	}
	if got.ID != user.ID {
		t.Errorf("got.ID = %d, want %d", got.ID, user.ID)
	}
	if got.Username != "admin" {
		t.Errorf("got.Username = %q, want %q", got.Username, "admin")
	}
	if !auth.CheckPassword(got.PasswordHash, "testpass") {
		t.Error("stored password hash should match original password")
	}
}

func TestUserStoreGetByUsernameNotFound(t *testing.T) {
	users, _ := newTestStores(t)
	_, err := users.GetByUsername(context.Background(), "nonexistent")
	if !errors.Is(err, auth.ErrUserNotFound) {
		t.Errorf("GetByUsername(missing) = %v, want ErrUserNotFound", err)
	}
}

func TestSessionStoreCRUD(t *testing.T) {
	users, sessions := newTestStores(t)
	ctx := context.Background()

	// Create a user first.
	hash, _ := auth.HashPassword("pass")
	user, err := users.Create(ctx, "sessionuser", hash)
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}

	// Create a session.
	token := auth.NewSessionToken()
	sess := auth.Session{
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(auth.SessionTTL),
	}
	if err := sessions.Create(ctx, sess); err != nil {
		t.Fatalf("Create session: %v", err)
	}

	// GetByToken should find the session.
	got, err := sessions.GetByToken(ctx, token)
	if err != nil {
		t.Fatalf("GetByToken: %v", err)
	}
	if got.Token != token {
		t.Errorf("Token = %q, want %q", got.Token, token)
	}
	if got.UserID != user.ID {
		t.Errorf("UserID = %d, want %d", got.UserID, user.ID)
	}

	// DeleteByToken should remove it.
	if err := sessions.DeleteByToken(ctx, token); err != nil {
		t.Fatalf("DeleteByToken: %v", err)
	}
	_, err = sessions.GetByToken(ctx, token)
	if !errors.Is(err, auth.ErrSessionExpired) {
		t.Errorf("GetByToken after delete = %v, want ErrSessionExpired", err)
	}
}

func TestSessionStoreDeleteExpired(t *testing.T) {
	users, sessions := newTestStores(t)
	ctx := context.Background()

	hash, _ := auth.HashPassword("pass")
	user, err := users.Create(ctx, "expireuser", hash)
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}

	// Create an already-expired session.
	expiredToken := auth.NewSessionToken()
	expired := auth.Session{
		Token:     expiredToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	if err := sessions.Create(ctx, expired); err != nil {
		t.Fatalf("Create expired session: %v", err)
	}

	// Create a valid session.
	validToken := auth.NewSessionToken()
	valid := auth.Session{
		Token:     validToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(auth.SessionTTL),
	}
	if err := sessions.Create(ctx, valid); err != nil {
		t.Fatalf("Create valid session: %v", err)
	}

	// DeleteExpired should remove only the expired session.
	if err := sessions.DeleteExpired(ctx); err != nil {
		t.Fatalf("DeleteExpired: %v", err)
	}

	// Expired session should be gone.
	_, err = sessions.GetByToken(ctx, expiredToken)
	if !errors.Is(err, auth.ErrSessionExpired) {
		t.Errorf("GetByToken(expired) = %v, want ErrSessionExpired", err)
	}

	// Valid session should still exist.
	got, err := sessions.GetByToken(ctx, validToken)
	if err != nil {
		t.Fatalf("GetByToken(valid): %v", err)
	}
	if got.Token != validToken {
		t.Errorf("valid session Token = %q, want %q", got.Token, validToken)
	}
}
