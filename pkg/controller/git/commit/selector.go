package commit

import (
	"context"
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
)

// Selector is an interface for selecting commits from a Git repository.
type Selector interface {
	// MatchesRef returns a boolean value indicating whether or not the Selector
	// would consider a commit referenced by the specified ref to be eligible for
	// selection.
	MatchesRef(string) bool
	// MatchesPaths returns a boolean value indicating whether or not the Selector
	// would consider a commit with the specified changed paths to be eligible for
	// selection.
	MatchesPaths([]string) bool
	// ListRefs returns the raw remote ref state relevant to the Selector's commit
	// selection strategy, obtained via a single git ls-remote round-trip without
	// cloning. Name-based filters (semver and/or regex) are applied; path filters
	// are not, as those are evaluated during Select. The result is suitable for
	// recording in Warehouse status and for a cheap equality check that detects
	// whether anything relevant has moved since the last discovery.
	ListRefs(context.Context) (*kargoapi.GitDiscoveryRefs, error)
	// Select selects images from a container image repository.
	Select(context.Context) ([]kargoapi.DiscoveredCommit, error)
}

// NewSelector returns some implementation of the Selector interface that
// selects commits from a Git repository based on the provided subscription.
func NewSelector(
	ctx context.Context,
	sub kargoapi.GitSubscription,
	creds *git.RepoCredentials,
) (Selector, error) {
	// Pick an appropriate Selector implementation based on the subscription
	// provided.
	reg, err := defaultSelectorRegistry.Get(ctx, sub)
	if err != nil {
		return nil, fmt.Errorf("error getting selector factory")
	}
	factory := reg.Value
	return factory(sub, creds)
}
