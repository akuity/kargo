package subscription

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/component"
	"github.com/akuity/kargo/pkg/credentials"
)

type (
	SubscriberPredicate = func(
		context.Context,
		kargoapi.RepoSubscription,
	) (bool, error)

	SubscriberFactory = func(
		context.Context,
		credentials.Database,
	) (Subscriber, error)

	SubscriberRegistration = component.PredicateBasedRegistration[
		kargoapi.RepoSubscription, // Arg to the predicate function
		SubscriberPredicate,       // Predicate function
		SubscriberFactory,         // Factory function
		SubscriberMetadata,        // Metadata
	]

	SubscriberRegistry = component.PredicateBasedRegistry[
		kargoapi.RepoSubscription, // Arg to the predicate function
		SubscriberPredicate,       // Predicate function
		SubscriberFactory,         // Factory function
		SubscriberMetadata,        // Metadata
	]
)

// SubscriberMetadata describes the subscriptions that a Subscriber handles.
type SubscriberMetadata struct {
	// Schema, if non-nil, is the JSON schema that the raw configuration of each
	// subscription handled by the Subscriber must conform to.
	Schema *Schema
}

func MustNewSubscriberRegistry(
	registrations ...SubscriberRegistration,
) SubscriberRegistry {
	r, err := component.NewPredicateBasedRegistry(registrations...)
	if err != nil {
		panic(err)
	}
	return r
}

var DefaultSubscriberRegistry = MustNewSubscriberRegistry()
