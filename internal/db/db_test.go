package db

import (
	"errors"
	"testing"
)

func TestParseDialect(t *testing.T) {
	cases := map[string]struct {
		in      string
		want    Dialect
		wantErr bool
	}{
		"postgres":     {"postgres", DialectPostgres, false},
		"sqlite":       {"sqlite", DialectSQLite, false},
		"unknown":      {"mysql", "", true},
		"empty":        {"", "", true},
		"uppercase pg": {"POSTGRES", DialectPostgres, false},
		"uppercase lt": {"SQLite", DialectSQLite, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := ParseDialect(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseDialect(%q) error = nil, want non-nil", tc.in)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseDialect(%q) error = %v, want nil", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("ParseDialect(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestErrNoRowsIsNotNil(t *testing.T) {
	if ErrNoRows == nil {
		t.Error("ErrNoRows must be defined")
	}
	if !errors.Is(ErrNoRows, ErrNoRows) {
		t.Error("ErrNoRows must be identifiable via errors.Is")
	}
}
