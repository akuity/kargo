package subscription

import (
	"context"

	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

type MockSubscriber struct {
	ApplySubscriptionDefaultsFn func(
		ctx context.Context,
		sub *kargoapi.RepoSubscription,
	) error
	ValidateSubscriptionFn func(
		ctx context.Context,
		f *field.Path,
		sub kargoapi.RepoSubscription,
	) field.ErrorList
	DiscoverArtifactsFn func(
		ctx context.Context,
		project string,
		sub kargoapi.RepoSubscription,
	) (any, error)
}

func (m *MockSubscriber) ApplySubscriptionDefaults(
	ctx context.Context,
	sub *kargoapi.RepoSubscription,
) error {
	return m.ApplySubscriptionDefaultsFn(ctx, sub)
}

func (m *MockSubscriber) ValidateSubscription(
	ctx context.Context,
	f *field.Path,
	sub kargoapi.RepoSubscription,
) field.ErrorList {
	return m.ValidateSubscriptionFn(ctx, f, sub)
}

func (m *MockSubscriber) DiscoverArtifacts(
	ctx context.Context,
	project string,
	sub kargoapi.RepoSubscription,
) (any, error) {
	if m.DiscoverArtifactsFn != nil {
		return m.DiscoverArtifactsFn(ctx, project, sub)
	}
	return nil, nil
}
