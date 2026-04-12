package downloadclient

import "fmt"

// Factory is a constructor function that returns a new, unconfigured
// DownloadClient.
type Factory func() DownloadClient

// Registry holds the set of known download client factories keyed by
// implementation name (e.g. "SABnzbd").
type Registry struct {
	factories map[string]Factory
}

// NewRegistry returns an empty Registry ready for use.
func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]Factory)}
}

// Register associates name with the given Factory. Panics if name is already
// registered so that duplicate registrations are caught at startup.
func (r *Registry) Register(name string, f Factory) {
	if _, exists := r.factories[name]; exists {
		panic(fmt.Sprintf("downloadclient: duplicate registration for %q", name))
	}
	r.factories[name] = f
}

// Get returns the Factory registered under name, or an error if not found.
func (r *Registry) Get(name string) (Factory, error) {
	f, ok := r.factories[name]
	if !ok {
		return nil, fmt.Errorf("downloadclient: no factory registered for %q", name)
	}
	return f, nil
}

// All returns a shallow copy of the factory map.
func (r *Registry) All() map[string]Factory {
	out := make(map[string]Factory, len(r.factories))
	for k, v := range r.factories {
		out[k] = v
	}
	return out
}
