package component

import (
	"context"
	"errors"
	"sync"
)

// listBasedRegistry is a simple implementation of the PredicateBasedRegistry
// interface that stores registrations in a slice and evaluates predicates in
// order when Get() is called.
type listBasedRegistry[PA any, P Predicate[PA], V, MD any] struct {
	registrations []PredicateBasedRegistration[PA, P, V, MD]
	mu            sync.RWMutex
}

// newListBasedRegistry returns a simple implementation of the
// PredicateBasedRegistry interface that stores registrations in a slice and
// evaluates predicates in order when Get() is called. Optional initial
// registrations may be provided when calling this function.
func newListBasedRegistry[PA any, P Predicate[PA], V, MD any](
	registrations ...PredicateBasedRegistration[PA, P, V, MD],
) (PredicateBasedRegistry[PA, P, V, MD], error) {
	r := &listBasedRegistry[PA, P, V, MD]{
		registrations: make(
			[]PredicateBasedRegistration[PA, P, V, MD],
			0,
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

// Register implements PredicateBasedRegistry.
func (r *listBasedRegistry[PA, P, V, MD]) Register(
	reg PredicateBasedRegistration[PA, P, V, MD],
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if reg.Predicate == nil {
		return errors.New("registration has nil predicate function")
	}
	r.registrations = append(r.registrations, reg)
	return nil
}

// MustRegister implements PredicateBasedRegistry.
func (r *listBasedRegistry[PA, P, V, MD]) MustRegister(
	reg PredicateBasedRegistration[PA, P, V, MD],
) {
	if err := r.Register(reg); err != nil {
		panic(err)
	}
}

// Get implements PredicateBasedRegistry.
func (r *listBasedRegistry[PA, P, V, MD]) Get(
	ctx context.Context,
	arg PA,
) (PredicateBasedRegistration[PA, P, V, MD], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var zeroReg PredicateBasedRegistration[PA, P, V, MD]
	for _, reg := range r.registrations {
		res, err := reg.Predicate(ctx, arg)
		if err != nil {
			return zeroReg, err
		}
		if res {
			return reg, nil
		}
	}
	return zeroReg, RegistrationNotFoundError{}
}
