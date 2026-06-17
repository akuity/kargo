package subscription

import (
	"context"

	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Subscriber is an interface to be implemented by components that can extract
// relevant subscription details from a kargoapi.RepoSubscription and use them
// to discover corresponding artifacts.
type Subscriber interface {
	// ApplySubscriptionDefaults applies default values to the provided
	// kargoapi.RepoSubscription.
	ApplySubscriptionDefaults(
		ctx context.Context,
		sub *kargoapi.RepoSubscription,
	) error
	// ValidateSubscription validates the provided kargoapi.RepoSubscription.
	ValidateSubscription(
		ctx context.Context,
		f *field.Path,
		sub kargoapi.RepoSubscription,
	) field.ErrorList
	// DiscoverArtifacts discovers artifacts according to the parameters of the
	// provided kargoapi.RepoSubscription. Implementations may return a value of
	// any type, but in practice, callers can only make sense of:
	// - kargoapi.ChartDiscoveryResult
	// - kargoapi.GitDiscoveryResult
	// - kargoapi.ImageDiscoveryResult
	// - kargoapi.DiscoveryResult
	//
	// last is the result this same subscription produced at the previous
	// successful discovery (of the same concrete type), or nil if there is none.
	// Implementations may use it to short-circuit expensive work when a cheap
	// check shows nothing relevant has changed; those that cannot simply ignore
	// it and return a freshly discovered result.
	DiscoverArtifacts(
		ctx context.Context,
		project string,
		sub kargoapi.RepoSubscription,
		last any,
	) (any, error)
}
