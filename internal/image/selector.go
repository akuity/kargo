package image

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Selector is an interface for selecting images from a container image
// repository.
type Selector interface {
	// MatchesTag returns a boolean value indicating whether or not the Selector
	// would consider an image with the specified tag eligible for selection.
	MatchesTag(string) bool
	// Select selects images from a container image repository.
	Select(context.Context) ([]kargoapi.DiscoveredImageReference, error)
}

// NewSelector returns some implementation of the Selector interface that
// selects images from a container image repository based on the provided
// subscription.
func NewSelector(
	sub kargoapi.ImageSubscription,
	creds *Credentials,
) (Selector, error) {
	// Pick an appropriate Selector implementation based on the subscription
	// provided.
	selectorFactory, err := selectorReg.getSelectorFactory(sub)
	if err != nil {
		return nil, err
	}
	return selectorFactory(sub, creds)
}
