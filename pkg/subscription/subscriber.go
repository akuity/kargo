package subscription

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Subscriber is an interface to be implemented by components that can extract
// relevant subscription details from a kargoapi.RepoSubscription and use them
// to discover corresponding artifacts.
type Subscriber interface {
	// DiscoverArtifacts discovers artifacts according to the parameters of the
	// provided kargoapi.RepoSubscription. Implementations may return a value of
	// any type, but in practice, callers can only make sense of:
	// - kargoapi.ChartDiscoveryResult
	// - kargoapi.GitDiscoveryResult
	// - kargoapi.ImageDiscoveryResult
	// - kargoapi.GenericDiscoveryResult
	DiscoverArtifacts(
		ctx context.Context,
		project string,
		sub kargoapi.RepoSubscription,
	) (any, error)
}
