package profiles

import "errors"

// ErrNotFound is returned by store methods when a requested row does not exist.
var ErrNotFound = errors.New("profiles: not found")
