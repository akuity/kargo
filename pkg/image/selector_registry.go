package image

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/component"
)

type (
	// selectorPredicate is a function which should return true if the provided
	// kargoapi.ImageSubscription indicates that a corresponding selectorFactory
	// function should be used to instantiate an appropriate implementation of
	// Selector.
	selectorPredicate = func(
		context.Context,
		kargoapi.ImageSubscription,
	) (bool, error)

	// selectorFactor is a function that returns an implementation of Selector.
	selectorFactory = func(
		kargoapi.ImageSubscription,
		*Credentials,
	) (Selector, error)

	// selectorRegistration associates a selectorPredicate with a selectorFactory.
	selectorRegistration = component.PredicateBasedRegistration[
		kargoapi.ImageSubscription, // Arg to the predicate function
		selectorPredicate,          // Predicate function
		selectorFactory,            // Factory function
		struct{},                   // This registry uses no metadata
	]
)

var defaultSelectorRegistry = component.MustNewPredicateBasedRegistry[
	kargoapi.ImageSubscription,
	selectorPredicate,
	selectorFactory,
	struct{},
]()
