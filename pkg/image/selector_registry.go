package image

import (
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// selectorPredicate is a function which should return true if the provided
// kargoapi.ImageSubscription indicates that a corresponding selectorFactory
// function should be used to instantiate an appropriate implementation of
// Selector.
type selectorPredicate func(kargoapi.ImageSubscription) bool

// selectorFactor is a function that returns an implementation of Selector.
type selectorFactory func(
	kargoapi.ImageSubscription,
	*Credentials,
) (Selector, error)

// selectorRegistration associates a selectorPredicate with a selectorFactory.
type selectorRegistration struct {
	predicate selectorPredicate
	factory   selectorFactory
}

// selectorRegistry is a map of selectorRegistrations indexed by the image
// selection strategy their factory functions instantiate.
type selectorRegistry map[kargoapi.ImageSelectionStrategy]selectorRegistration

// register is invoked once for each implementation of
// Selector upon package initialization to associate a
// selectorPredicate with a selectorFactory.
func (s selectorRegistry) register(
	strategy kargoapi.ImageSelectionStrategy,
	registration selectorRegistration,
) {
	if _, alreadyRegistered := s[strategy]; alreadyRegistered {
		panic(
			fmt.Sprintf("Selector for strategy %q already registered", strategy),
		)
	}
	s[strategy] = registration
}

// getSelectorFactory retrieves a selectorFactory able to instantiate a Selector
// of the proper type based on the provided kargoapi.ImageSubscription. If no
// such factory can be found, an error is returned.
func (s selectorRegistry) getSelectorFactory(
	sub kargoapi.ImageSubscription,
) (selectorFactory, error) {
	for _, registration := range s {
		if registration.predicate(sub) {
			return registration.factory, nil
		}
	}
	return nil, fmt.Errorf(
		"ImageSubscription has matches no known Selector type",
	)
}

// selectorReg is the registry of selectorRegistrations.
var selectorReg = selectorRegistry{}
