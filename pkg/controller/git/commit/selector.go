package commit

import (
	"context"
	"errors"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
)

// Selector is an interface for selecting commits from a Git repository.
type Selector interface {
	// MatchesRef returns a boolean value indicating whether or not the Selector
	// would consider a commit referenced by the specified ref to be eligible for
	// selection.
	MatchesRef(string) bool
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
	reg, found, err := defaultSelectorRegistry.Get(ctx, sub)
	if err != nil {
		return nil, err
	}
	if !found {
		// This shouldn't happen because the API doesn't allow opting for any
		// selector for which for which we've not registered an implementation.
		return nil, errors.New("no selector found for subscription")
	}
	factory := reg.Value
	return factory(sub, creds)
}
