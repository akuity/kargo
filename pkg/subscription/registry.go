package subscription

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/component"
	gitpkg "github.com/akuity/kargo/pkg/controller/git"
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
		struct{},                  // This registry uses no metadata
	]

	SubscriberRegistry = component.PredicateBasedRegistry[
		kargoapi.RepoSubscription, // Arg to the predicate function
		SubscriberPredicate,       // Predicate function
		SubscriberFactory,         // Factory function
		struct{},                  // This registry uses no metadata
	]
)

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

// NewSubscriberRegistryWithGitCache creates a new subscriber registry that
// includes all the default subscriber types (git, image, chart) but uses
// a cache-aware git subscriber backed by the provided RepoCache.
func NewSubscriberRegistryWithGitCache(repoCache *gitpkg.RepoCache) SubscriberRegistry {
	registry := MustNewSubscriberRegistry()

	// Register cache-aware git subscriber
	registry.MustRegister(GitSubscriberRegistrationWithCache(repoCache))

	// Register image subscriber
	registry.MustRegister(SubscriberRegistration{
		Predicate: func(_ context.Context, sub kargoapi.RepoSubscription) (bool, error) {
			return sub.Image != nil, nil
		},
		Value: newImageSubscriber,
	})

	// Register chart subscriber
	registry.MustRegister(SubscriberRegistration{
		Predicate: func(_ context.Context, sub kargoapi.RepoSubscription) (bool, error) {
			return sub.Chart != nil, nil
		},
		Value: newChartSubscriber,
	})

	return registry
}
