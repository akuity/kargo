package warehouses

import (
	"context"
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/logging"
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

		logger := logging.LoggerFromContext(ctx).WithField("repoURL", sub.RepoURL)
		if sub.Name != "" {
			logger = logger.WithField("chart", sub.Name)
		}

		creds, ok, err := r.credentialsDB.Get(ctx, namespace, credentials.TypeHelm, sub.RepoURL)
		if err != nil {
			return nil, fmt.Errorf(
				"error obtaining credentials for chart repository %q: %w",
				sub.RepoURL,
				err,
			)
		}
		var helmCreds *helm.Credentials
		if ok {
			helmCreds = &helm.Credentials{
				Username: creds.Username,
				Password: creds.Password,
			}
			logger.Debug("obtained credentials for chart repo")
		} else {
			logger.Debug("found no credentials for chart repo")
		}

		versions, err := r.discoverChartVersionsFn(ctx, sub.RepoURL, sub.Name, sub.SemverConstraint, helmCreds)
		if err != nil {
			if sub.Name == "" {
				return nil, fmt.Errorf(
					"error discovering latest suitable chart versions in repository %q: %w",
					sub.RepoURL,
					err,
				)
			}
			return nil, fmt.Errorf(
				"error discovering latest suitable chart versions for chart %q in repository %q: %w",
				sub.Name,
				sub.RepoURL,
				err,
			)
		}

		if len(versions) == 0 {
			logger.Debug("discovered no suitable chart versions")
			results = append(results, kargoapi.ChartDiscoveryResult{
				RepoURL:          sub.RepoURL,
				Name:             sub.Name,
				SemverConstraint: sub.SemverConstraint,
			})
			continue
		}

		logger.Debugf("discovered %d suitable chart versions", len(versions))
		results = append(results, kargoapi.ChartDiscoveryResult{
			RepoURL:          sub.RepoURL,
			Name:             sub.Name,
			SemverConstraint: sub.SemverConstraint,
			Versions:         trimSlice(versions, 20),
		})
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
