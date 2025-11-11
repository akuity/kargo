package commit

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/component"
	"github.com/akuity/kargo/pkg/controller/git"
)

type (
	// selectorPredicate is a function which should return true if the provided
	// kargoapi.GitSubscription indicates that a corresponding selectorFactory
	// function should be used to instantiate an appropriate implementation of
	// Selector.
	selectorPredicate = func(
		context.Context,
		kargoapi.GitSubscription,
	) (bool, error)

	// selectorFactory is a function that returns an implementation of Selector.
	selectorFactory = func(
		kargoapi.GitSubscription,
		*git.RepoCredentials,
	) (Selector, error)

	// selectorRegistration associates a selectorPredicate with a selectorFactory.
	selectorRegistration = component.PredicateBasedRegistration[
		kargoapi.GitSubscription, // Arg to the predicate function
		selectorPredicate,        // Predicate function
		selectorFactory,          // Factory function
		struct{},                 // This registry uses no metadata
	]
)

var defaultSelectorRegistry = component.MustNewPredicateBasedRegistry[
	kargoapi.GitSubscription,
	selectorPredicate,
	selectorFactory,
	struct{},
]()
