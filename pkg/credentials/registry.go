package credentials

import (
	"context"

	"github.com/akuity/kargo/pkg/component"
)

type (
	ProviderPredicate = func(context.Context, Request) (bool, error)

	ProviderRegistration = component.PredicateBasedRegistration[
		Request,           // Arg to the predicate function
		ProviderPredicate, // Predicate function
		Provider,          // Type stored in the registry
		struct{},          // This registry uses no metadata
	]

	ProviderRegistry = component.PredicateBasedRegistry[
		Request,           // Arg to the predicate function
		ProviderPredicate, // Predicate function
		Provider,          // Type stored in the registry
		struct{},          // This registry uses no metadata
	]
)

func MustNewProviderRegistry(
	registrations ...ProviderRegistration,
) ProviderRegistry {
	r, err := component.NewPredicateBasedRegistry(registrations...)
	if err != nil {
		panic(err)
	}
	return r
}

var DefaultProviderRegistry = MustNewProviderRegistry()
