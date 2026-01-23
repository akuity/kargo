package subscription

import (
	"context"
	"fmt"
	"regexp"
	"strings"

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
		Value: newChartSubscriber,
	})
}

// chartSubscriber is an implementation of the Subscriber interface that
// discovers Helm chart versions from a Helm chart repository.
type chartSubscriber struct {
	credentialsDB credentials.Database
}

// newChartSubscriber returns an implementation of the Subscriber interface that
// discovers Helm chart versions from a Helm chart repository.
func newChartSubscriber(
	_ context.Context,
	credentialsDB credentials.Database,
) (Subscriber, error) {
	return &chartSubscriber{credentialsDB: credentialsDB}, nil
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

var helmRepoURLRegex = regexp.MustCompile(`^(((https?)|(oci))://)([\w\d\.\-]+)(:[\d]+)?(/.*)*$`)

// ValidateSubscription implements Subscriber.
func (c *chartSubscriber) ValidateSubscription(
	_ context.Context,
	f *field.Path,
	s kargoapi.RepoSubscription,
) field.ErrorList {
	// TODO(krancour): Longer term, we might want to start doing this with JSON
	// schema.

	sub := s.Chart
	var errs field.ErrorList

	// Validate RepoURL: MinLength=1, Pattern (Helm repo URL regex)
	if err := validation.MinLength(f.Child("repoURL"), sub.RepoURL, 1); err != nil {
		errs = append(errs, err)
	}
	if !helmRepoURLRegex.MatchString(sub.RepoURL) {
		errs = append(errs, field.Invalid(
			f.Child("repoURL"),
			sub.RepoURL,
			"must be a valid Helm repository URL",
		))
	}

	// Validate Name based on RepoURL
	if strings.HasPrefix(sub.RepoURL, "oci://") && sub.Name != "" {
		errs = append(
			errs,
			field.Invalid(
				f.Child("name"),
				sub.Name,
				"must be empty if repoURL starts with oci://",
			),
		)
	}
	isHTTP := strings.HasPrefix(sub.RepoURL, "http://") || strings.HasPrefix(sub.RepoURL, "https://")
	if isHTTP && sub.Name == "" {
		errs = append(
			errs,
			field.Invalid(
				f.Child("name"),
				sub.Name,
				"must be non-empty if repoURL starts with http:// or https://",
			),
		)
	}

	// Validate SemverConstraint
	if err := validation.SemverConstraint(
		f.Child("semverConstraint"),
		sub.SemverConstraint,
	); err != nil {
		errs = append(errs, err)
	}

	// Validate DiscoveryLimit: Minimum=1, Maximum=100
	if sub.DiscoveryLimit < 1 {
		errs = append(errs, field.Invalid(f.Child("discoveryLimit"), sub.DiscoveryLimit, "must be >= 1"))
	} else if sub.DiscoveryLimit > 100 {
		errs = append(errs, field.Invalid(f.Child("discoveryLimit"), sub.DiscoveryLimit, "must be <= 100"))
	}

	return errs
}

// DiscoverArtifacts implement Subscriber.
func (c *chartSubscriber) DiscoverArtifacts(
	ctx context.Context,
	project string,
	sub kargoapi.RepoSubscription,
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

	selector, err := chart.NewSelector(*chartSub, helmCreds)
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
