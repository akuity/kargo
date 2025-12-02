package datesub

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/component"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/subscription"
)

const subscriberTypeDemo = "demo"

func init() {
	// Register a demo credentials provider.
	provider := &credsProvider{}
	credentials.DefaultProviderRegistry.MustRegister(
		credentials.ProviderRegistration{
			Predicate: provider.Supports,
			Value:     provider,
		},
	)

	// Register the demo subscriber.
	var subscriber subscription.Subscriber
	var once sync.Once
	subscription.DefaultSubscriberRegistry.MustRegister(
		subscription.SubscriberRegistration{
			Predicate: func(
				_ context.Context,
				sub kargoapi.RepoSubscription,
			) (bool, error) {
				return sub.Subscription != nil &&
					sub.Subscription.SubscriptionType == subscriberTypeDemo, nil
			},
			// This factory function returns a single, stateful, but concurrency-safe
			// subscriber that's initialized just once the first time the factory
			// function is invoked.
			Value: func(
				ctx context.Context,
				credsDB credentials.Database,
			) (subscription.Subscriber, error) {
				once.Do(func() {
					subscriber = newSubscriber(ctx, credsDB)
				})
				return subscriber, nil
			},
		},
	)
}

// credsProvider is an implementation of credentials.Provider for "credentials"
// (they're empty) of type "demo". This exists for no reason other than to
// demonstrate that implementations of subscription.Subscriber with a need to
// retrieve some previously-unsupported type of credentials have a means to do
// so.
type credsProvider struct{}

// Supports implements credentials.Provider.
func (c *credsProvider) Supports(
	_ context.Context,
	req credentials.Request,
) (bool, error) {
	return req.Type == subscriberTypeDemo, nil
}

// GetCredentials implements credentials.Provider.
func (c *credsProvider) GetCredentials(
	context.Context,
	credentials.Request,
) (*credentials.Credentials, error) {
	return &credentials.Credentials{}, nil
}

// subscriber is an implementation of subscription.Subscriber for subscriptions
// of type "demo". It is stateful, but concurrency-safe. Internally, it
// maintains a collection of dummy artifacts that grows by one every time
// discovery is run (which is quite convenient for demo purposes).
type subscriber struct {
	latestVersion *semver.Version
	artifacts     []kargoapi.ArtifactReference
	credsDB       credentials.Database
	mu            sync.Mutex
}

// newSubscriber initialized and returns a demo implementation of the
// subscription.Subscriber interface.
func newSubscriber(_ context.Context,
	credsDB credentials.Database,
) subscription.Subscriber {
	return &subscriber{
		artifacts: []kargoapi.ArtifactReference{},
		credsDB:   credsDB,
	}
}

// DiscoverArtifacts implements subscription.Subscriber. It grows an internal
// collection of dummy artifacts by one each time it is invoked and returns a
// GenericDiscoveryResult. The maximum number of artifact references in the
// result is constrained by the DiscoveryLimit attribute of the subscription's
// configuration.
func (s *subscriber) DiscoverArtifacts(
	ctx context.Context,
	project string,
	sub kargoapi.RepoSubscription,
) (any, error) {
	// We're not actually doing anything with these dummy credentials, except
	// proving that we can find them.
	_, err := s.credsDB.Get(ctx, project, subscriberTypeDemo, sub.Subscription.Name)
	if err != nil {
		if !component.IsNotFoundError(err) {
			return nil, fmt.Errorf(
				"error finding subscriber for subscription type %q",
				subscriberTypeDemo,
			)
		}
	}

	// Unpack the subscription's configuration. This would be a good place to
	// validate the configuration if applicable.
	//
	// Note(krancour): We could potentially do schema-based validation of all
	// generic subscriptions by incorporating it into the existing validating
	// webhook for Warehouse resources. I've elected not to do that just yet and,
	// instead, handle generic subscriptions' opaque configuration in the same way
	// we handle the opaque configuration of a promotion step (i.e. validate just
	// prior to use). The reason promotion step configuration is evaluated in that
	// manner is that it supports expressions and and therefore cannot be
	// validated until expressions have been evaluated, which can only be done
	// within the context of an actual promotion. It's not yet clear whether some
	// kind of expression support may be required here as well, which could force
	// us to not to use a validating webhook.
	//
	// I believe we'll discover and refine exactly how we want to approach
	// validation after enabling a few new subscription types in EE.
	cfg := struct {
		Message string `json:"message,omitempty"`
	}{}
	if sub.Subscription.Config != nil {
		if err := json.Unmarshal(sub.Subscription.Config.Raw, &cfg); err != nil {
			return nil, err
		}
	}

	const defaultDiscoveryLimit = 20
	discoveryLimit := sub.Subscription.DiscoveryLimit
	if discoveryLimit == 0 {
		discoveryLimit = defaultDiscoveryLimit
	}

	// Bump the internal version number.
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.latestVersion == nil {
		s.latestVersion = semver.MustParse("v1.0.0")
	} else {
		*s.latestVersion = s.latestVersion.IncMinor()
	}

	// Grow the internal collection of artifacts.
	s.artifacts = slices.Insert(s.artifacts, 0, kargoapi.ArtifactReference{
		ArtifactType:     subscriberTypeDemo,
		SubscriptionName: sub.Subscription.Name,
		Version:          s.latestVersion.String(),
		// The details are opaque to the rest of Kargo in the same way the
		// subscription's configuration is.
		Metadata: &v1.JSON{Raw: json.RawMessage(
			fmt.Sprintf(
				`{"discoveredAt":%q, "message":%q}`,
				time.Now().String(), cfg.Message,
			),
		)},
	})

	// Trim the internal collection of artifacts if it has grown very large.
	const maxCollectionSize = 100
	if len(s.artifacts) > maxCollectionSize {
		s.artifacts = s.artifacts[:maxCollectionSize]
	}

	artifacts := s.artifacts
	if len(artifacts) > int(discoveryLimit) {
		artifacts = artifacts[:discoveryLimit]
	}

	return kargoapi.DiscoveryResult{
		SubscriptionName:   sub.Subscription.Name,
		ArtifactReferences: artifacts,
	}, nil
}
