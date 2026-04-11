package hostconfig

import (
	"testing"
	"time"
)

func TestHostConfigFields(t *testing.T) {
	hc := HostConfig{
		APIKey:         "abc123",
		AuthMode:       "forms",
		MigrationState: "clean",
		CreatedAt:      time.Unix(0, 0),
		UpdatedAt:      time.Unix(0, 0),
	}
	if hc.APIKey != "abc123" {
		t.Errorf("APIKey = %q", hc.APIKey)
	}
}

func TestNewAPIKeyIsNonEmpty(t *testing.T) {
	k := NewAPIKey()
	if len(k) < 32 {
		t.Errorf("API key length = %d, want >= 32", len(k))
	}
}

func TestNewAPIKeyIsUnique(t *testing.T) {
	seen := make(map[string]struct{})
	for i := 0; i < 10; i++ {
		k := NewAPIKey()
		if _, ok := seen[k]; ok {
			t.Errorf("duplicate API key generated: %q", k)
		}
		seen[k] = struct{}{}
	}
}
