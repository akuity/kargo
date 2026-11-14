package subscription

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/helm"
	"github.com/akuity/kargo/pkg/helm/chart"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/validation"
)

func init() {
	DefaultSubscriberRegistry.MustRegister(SubscriberRegistration{
		Predicate: func(
			_ context.Context,
			sub kargoapi.RepoSubscription,
		) (bool, error) {
			return sub.Chart != nil, nil
		},
		Value:    newChartSubscriber,
		Metadata: SubscriberMetadata{Schema: mustLoadSchema("chart")},
	})
}

// chartSubscriber is an implementation of the Subscriber interface that
// discovers Helm chart versions from a Helm chart repository.
type chartSubscriber struct {
	credentialsDB credentials.Database

	// newSelectorFn constructs the chart Selector for a subscription. It is a
	// field so tests can substitute a fake Selector for the real one.
	newSelectorFn func(
		ctx context.Context,
		sub kargoapi.ChartSubscription,
		creds *helm.Credentials,
	) (chart.Selector, error)
}

// newChartSubscriber returns an implementation of the Subscriber interface that
// discovers Helm chart versions from a Helm chart repository.
func newChartSubscriber(
	_ context.Context,
	credentialsDB credentials.Database,
) (Subscriber, error) {
	return &chartSubscriber{
		credentialsDB: credentialsDB,
		newSelectorFn: chart.NewSelector,
	}, nil
}

// ApplySubscriptionDefaults implements Subscriber.
func (c *chartSubscriber) ApplySubscriptionDefaults(
	_ context.Context,
	sub *kargoapi.RepoSubscription,
) error {
	if sub == nil || sub.Chart == nil {
		return nil
	}
	if sub.Chart != nil && sub.Chart.DiscoveryLimit == 0 {
		sub.Chart.DiscoveryLimit = 20
	}
	return nil
}

// ValidateSubscription implements Subscriber. The subscription's schema
// covers everything else.
func (c *chartSubscriber) ValidateSubscription(
	_ context.Context,
	f *field.Path,
	s kargoapi.RepoSubscription,
) field.ErrorList {
	if err := validation.SemverConstraint(
		f.Child("semverConstraint"),
		s.Chart.SemverConstraint,
	); err != nil {
		return field.ErrorList{err}
	}
	return nil
}

// DiscoverArtifacts implement Subscriber.
func (c *chartSubscriber) DiscoverArtifacts(
	ctx context.Context,
	project string,
	sub kargoapi.RepoSubscription,
	_ any,
) (any, error) {
	chartSub := sub.Chart

	if chartSub == nil {
		return nil, nil
	}

	logger := logging.LoggerFromContext(ctx).WithValues("repo", chartSub.RepoURL)
	if chartSub.Name != "" {
		logger = logger.WithValues("chart", chartSub.Name)
	}

	// Obtain credentials for the chart repository.
	creds, err := c.credentialsDB.Get(
		ctx,
		project,
		credentials.TypeHelm,
		chartSub.RepoURL,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"error obtaining credentials for chart repository %q: %w",
			chartSub.RepoURL, err,
		)
	}
	var helmCreds *helm.Credentials
	if creds != nil {
		helmCreds = &helm.Credentials{
			Username: creds.Username,
			Password: creds.Password,
		}
		logger.Debug("obtained credentials for chart repo")
	} else {
		logger.Debug("found no credentials for chart repo")
	}

	selector, err := c.newSelectorFn(ctx, *chartSub, helmCreds)
	if err != nil {
		return nil, fmt.Errorf(
			"error obtaining selector for chart versions from helm chart repo %q: %w",
			chartSub.RepoURL, err,
		)
	}
	versions, err := selector.Select(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"error discovering chart versions from helm chart repo %q: %w",
			chartSub.RepoURL, err,
		)
	}
	if len(versions) == 0 {
		logger.Debug("discovered no chart versions")
	} else {
		logger.Debug("discovered chart versions", "count", len(versions))
	}

	return kargoapi.ChartDiscoveryResult{
		RepoURL:          chartSub.RepoURL,
		Name:             chartSub.Name,
		SemverConstraint: chartSub.SemverConstraint,
		Versions:         trimSlice(versions, int(chartSub.DiscoveryLimit)),
		SubscriptionName: sub.Name,
	}, nil
}

// trimSlice returns a slice of any type with a maximum length of limit.
// If the input slice is shorter than limit or limit is less than or equal to
// zero, the input slice is returned unmodified.
func trimSlice[T any](slice []T, limit int) []T {
	if limit <= 0 || len(slice) <= limit {
		return slice
	}
	return slice[:limit]
}
