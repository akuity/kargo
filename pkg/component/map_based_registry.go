package component

import (
	"fmt"
	"sync"
)

// mapBasedRegistry is a simple implementation of the NameBasedRegistry
// interface that stores registrations in a map.
type mapBasedRegistry[V, MD any] struct {
	opts          NameBasedRegistryOptions
	registrations map[string]NameBasedRegistration[V, MD]
	mu            sync.RWMutex
}

// newMapBasedRegistry returns a simple implementation of the NameBasedRegistry
// interface that stores registrations in a map.
func newMapBasedRegistry[V, MD any](
	opts *NameBasedRegistryOptions,
	registrations ...NameBasedRegistration[V, MD],
) (NameBasedRegistry[V, MD], error) {
	if opts == nil {
		opts = &NameBasedRegistryOptions{}
	}
	r := &mapBasedRegistry[V, MD]{
		opts: *opts,
		registrations: make(
			map[string]NameBasedRegistration[V, MD],
			len(registrations),
		),
	}
	for _, reg := range registrations {
		if err := r.Register(reg); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Register implements NameBasedRegistry.
func (r *mapBasedRegistry[V, MD]) Register(
	reg NameBasedRegistration[V, MD],
) error {
	if reg.Name == "" {
		return fmt.Errorf("registration name cannot be empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.registrations[reg.Name]; exists && !r.opts.AllowOverwriting {
		return fmt.Errorf(
			"cannot overwrite registration with name %q; registry does not "+
				"permit overwriting existing registrations", reg.Name,
		)
	}
	r.registrations[reg.Name] = reg
	return nil
}

// MustRegister implements NameBasedRegistry.
func (r *mapBasedRegistry[V, MD]) MustRegister(
	reg NameBasedRegistration[V, MD],
) {
	if err := r.Register(reg); err != nil {
		panic(err)
	}
}

// Get implements NameBasedRegistry.
func (r *mapBasedRegistry[V, MD]) Get(
	name string,
) (NameBasedRegistration[V, MD], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	reg, found := r.registrations[name]
	if !found {
		return reg, NamedRegistrationNotFoundError{}
	}
	return reg, nil
}
