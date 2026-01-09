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
	DiscoverArtifacts(
		ctx context.Context,
		project string,
		sub kargoapi.RepoSubscription,
	) (any, error)
}
