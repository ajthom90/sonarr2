// Package providers defines the base interfaces and schema utilities shared
// by all provider kinds (indexers, download clients, etc.).
package providers

import "context"

// Provider is the base interface all provider kinds extend.
type Provider interface {
	Implementation() string
	DefaultName() string
	Settings() any
	Test(ctx context.Context) error
}
