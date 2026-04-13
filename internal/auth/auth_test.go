package auth

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("mysecretpassword")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !CheckPassword(hash, "mysecretpassword") {
		t.Error("correct password should match")
	}
	if CheckPassword(hash, "wrongpassword") {
		t.Error("wrong password should not match")
	}
}

func TestNewSessionToken(t *testing.T) {
	t1 := NewSessionToken()
	t2 := NewSessionToken()
	if len(t1) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("token length = %d, want 64", len(t1))
	}
	if t1 == t2 {
		t.Error("tokens should be unique")
	}
}
