package commit

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
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
	sub kargoapi.GitSubscription,
	creds *git.RepoCredentials,
) (Selector, error) {
	// Pick an appropriate Selector implementation based on the subscription
	// provided.
	selectorFactory, err := selectorReg.getSelectorFactory(sub)
	if err != nil {
		return nil, err
	}
	return selectorFactory(sub, creds)
}
