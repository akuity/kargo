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
// WebhookReceiver. Such functions MUST NOT access fields of or invoke methods
// of the provided client.Client because the registration process will invoke
// this factory function and pass nil for the client.Client parameter.
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
var webhookReceiverRegistry = map[string]webhookReceiverRegistration{}

// registerWebhookReceiver is invoked once for each implementation of
// WebhookReceiver upon package initialization to associate a
// webhookReceiverPredicate with a webhookReceiverFactory.
func registerWebhookReceiver(
	predicate webhookReceiverPredicate,
	factory webhookReceiverFactory,
) {
	if predicate == nil {
		panic("predicate cannot be nil")
	}
	if factory == nil {
		panic("factory cannot be nil")
	}
	receiver := factory(nil, "", kargoapi.WebhookReceiverConfig{})
	receiverType := receiver.getReceiverType()
	if _, alreadyRegistered := webhookReceiverRegistry[receiverType]; alreadyRegistered {
		panic(fmt.Sprintf("WebhookReceiver %q already registered", receiverType))
	}
	webhookReceiverRegistry[receiverType] = webhookReceiverRegistration{
		predicate: predicate,
		factory:   factory,
	}
}
