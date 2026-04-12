package commands

import "fmt"

// Registry maps command names to Handlers.
type Registry struct {
	handlers map[string]Handler
}

func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]Handler)}
}

func (r *Registry) Register(name string, h Handler) {
	r.handlers[name] = h
}

func (r *Registry) Get(name string) (Handler, error) {
	h, ok := r.handlers[name]
	if !ok {
		return nil, fmt.Errorf("commands: no handler registered for %q", name)
	}
	return h, nil
}
