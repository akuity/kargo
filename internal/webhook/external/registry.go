package external

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// webhookReceiverPredicate is a function which should return true if the
// provided kargoapi.WebhookReceiverConfig indicates that a corresponding
// webhookReceiverFactory function should be used to instantiate an appropriate
// implementation of WebhookReceiver.
type webhookReceiverPredicate func(kargoapi.WebhookReceiverConfig) bool

// webhookReceiverFactory is a function that returns an implementation of
// WebhookReceiver.
type webhookReceiverFactory func(
	c client.Client,
	project string,
	cfg kargoapi.WebhookReceiverConfig,
) WebhookReceiver

// webhookReceiverRegistration associates a webhookReceiverPredicate with a
// webhookReceiverFactory.
type webhookReceiverRegistration struct {
	predicate webhookReceiverPredicate
	factory   webhookReceiverFactory
}

// webhookReceiverRegistry is a map of webhookReceiverRegistrations indexed by
// the names of the WebhookReceiver implementations their factory functions
// instantiate.
type webhookReceiverRegistry map[string]webhookReceiverRegistration

// register is invoked once for each implementation of
// WebhookReceiver upon package initialization to associate a
// webhookReceiverPredicate with a webhookReceiverFactory.
func (w webhookReceiverRegistry) register(
	receiverType string,
	registration webhookReceiverRegistration,
) {
	if _, alreadyRegistered := registry[receiverType]; alreadyRegistered {
		panic(
			fmt.Sprintf("WebhookReceiver type %q already registered", receiverType),
		)
	}
	registry[receiverType] = registration
}

// getReceiverFactory retrieves a webhookReceiverFactory able to instantiate a
// WebhookReceiver of the proper type based on the provided
// kargoapi.WebhookReceiverConfig. If no such factory can be found, an error is
// returned.
func (w webhookReceiverRegistry) getReceiverFactory(
	cfg kargoapi.WebhookReceiverConfig,
) (webhookReceiverFactory, error) {
	for _, registration := range registry {
		if registration.predicate(cfg) {
			return registration.factory, nil
		}
	}
	return nil, fmt.Errorf(
		"WebhookReceiverConfig has no configuration for a known receiver type",
	)
}

// registry is the registry of webhookReceiverRegistrations.
var registry = webhookReceiverRegistry{}
