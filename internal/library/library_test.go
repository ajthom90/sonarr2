package library

import (
	"errors"
	"testing"
)

func TestErrNotFoundIsDefined(t *testing.T) {
	if ErrNotFound == nil {
		t.Error("ErrNotFound must not be nil")
	}
	if !errors.Is(ErrNotFound, ErrNotFound) {
		t.Error("ErrNotFound must be identifiable via errors.Is")
	}
}
