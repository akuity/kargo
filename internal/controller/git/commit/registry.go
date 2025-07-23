package commit

import (
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
)

// selectorPredicate is a function which should return true if the provided
// kargoapi.GitSubscription indicates that a corresponding selectorFactory
// function should be used to instantiate an appropriate implementation of
// Selector.
type selectorPredicate func(kargoapi.GitSubscription) bool

// selectorFactor is a function that returns an implementation of Selector.
type selectorFactory func(
	kargoapi.GitSubscription,
	*git.RepoCredentials,
) (Selector, error)

// selectorRegistration associates a selectorPredicate with a selectorFactory.
type selectorRegistration struct {
	predicate selectorPredicate
	factory   selectorFactory
}

// selectorRegistry is a map of selectorRegistrations indexed by the commit
// selection strategy their factory functions instantiate.
type selectorRegistry map[kargoapi.CommitSelectionStrategy]selectorRegistration

// register is invoked once for each implementation of
// Selector upon package initialization to associate a
// selectorPredicate with a selectorFactory.
func (s selectorRegistry) register(
	strategy kargoapi.CommitSelectionStrategy,
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
// of the proper type based on the provided kargoapi.GitSubscription. If no
// such factory can be found, an error is returned.
func (s selectorRegistry) getSelectorFactory(
	sub kargoapi.GitSubscription,
) (selectorFactory, error) {
	for _, registration := range s {
		if registration.predicate(sub) {
			return registration.factory, nil
		}
	}
	return nil, fmt.Errorf("GitSubscription matches no known Selector type")
}

// selectorReg is the registry of selectorRegistrations.
var selectorReg = selectorRegistry{}
