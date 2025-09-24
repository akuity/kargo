package warehouses

import (
	"context"
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/helm"
	"github.com/akuity/kargo/pkg/helm/chart"
	"github.com/akuity/kargo/pkg/logging"
)

func (r *reconciler) discoverCharts(
	ctx context.Context,
	namespace string,
	subs []kargoapi.RepoSubscription,
) ([]kargoapi.ChartDiscoveryResult, error) {
	results := make([]kargoapi.ChartDiscoveryResult, 0, len(subs))

	for _, s := range subs {
		if s.Chart == nil {
			continue
		}

		sub := s.Chart

		logger := logging.LoggerFromContext(ctx).WithValues("repoURL", sub.RepoURL)
		if sub.Name != "" {
			logger = logger.WithValues("chart", sub.Name)
		}

		creds, err := r.credentialsDB.Get(ctx, namespace, credentials.TypeHelm, sub.RepoURL)
		if err != nil {
			return nil, fmt.Errorf(
				"error obtaining credentials for chart repository %q: %w",
				sub.RepoURL,
				err,
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

		selector, err := chart.NewSelector(*s.Chart, helmCreds)
		if err != nil {
			return nil, fmt.Errorf(
				"error obtaining selector for chart versions from helm chart repo %q: %w",
				sub.RepoURL, err,
			)
		}
		versions, err := selector.Select(ctx)
		if err != nil {
			return nil, fmt.Errorf(
				"error discovering chart versions from helm chart repo %q: %w",
				sub.RepoURL, err,
			)
		}

		if len(versions) == 0 {
			results = append(results, kargoapi.ChartDiscoveryResult{
				RepoURL:          sub.RepoURL,
				Name:             sub.Name,
				SemverConstraint: sub.SemverConstraint,
			})
			logger.Debug("discovered no chart versions")
			continue
		}

		results = append(results, kargoapi.ChartDiscoveryResult{
			RepoURL:          sub.RepoURL,
			Name:             sub.Name,
			SemverConstraint: sub.SemverConstraint,
			Versions:         trimSlice(versions, int(sub.DiscoveryLimit)),
		})
		logger.Debug(
			"discovered chart versions",
			"count", len(versions),
		)
	}

	return results, nil
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
