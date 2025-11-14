package external

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/component"
)

type (
	// webhookReceiverPredicate is a function which should return true if the
	// provided kargoapi.WebhookReceiverConfig indicates that a corresponding
	// webhookReceiverFactory function should be used to instantiate an
	// appropriate implementation of WebhookReceiver.
	webhookReceiverPredicate = func(
		context.Context,
		kargoapi.WebhookReceiverConfig,
	) (bool, error)

	// webhookReceiverFactory is a function that returns an implementation of
	// WebhookReceiver.
	webhookReceiverFactory = func(
		c client.Client,
		project string,
		cfg kargoapi.WebhookReceiverConfig,
	) WebhookReceiver

	// webhookReceiverRegistration associates a webhookReceiverPredicate with a
	// webhookReceiverFactory.
	webhookReceiverRegistration = component.PredicateBasedRegistration[
		kargoapi.WebhookReceiverConfig, // Arg to the predicate function
		webhookReceiverPredicate,       // Predicate function
		webhookReceiverFactory,         // Factory function
		struct{},                       // This registry uses no metadata
	]
)

var defaultWebhookReceiverRegistry = component.MustNewPredicateBasedRegistry[
	kargoapi.WebhookReceiverConfig,
	webhookReceiverPredicate,
	webhookReceiverFactory,
	struct{},
]()
