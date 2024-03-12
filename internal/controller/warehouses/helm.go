package warehouses

import (
	"context"
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/logging"
)

func (r *reconciler) selectCharts(
	ctx context.Context,
	namespace string,
	subs []kargoapi.RepoSubscription,
) ([]kargoapi.Chart, error) {
	charts := make([]kargoapi.Chart, 0, len(subs))

	for _, s := range subs {
		if s.Chart == nil {
			continue
		}

		sub := s.Chart

		logger := logging.LoggerFromContext(ctx).WithField("repoURL", sub.RepoURL)
		if sub.Name != "" {
			logger = logger.WithField("chart", sub.Name)
		}

		creds, ok, err :=
			r.credentialsDB.Get(ctx, namespace, credentials.TypeHelm, sub.RepoURL)
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

		vers, err := r.selectChartVersionFn(
			ctx,
			sub.RepoURL,
			sub.Name,
			sub.SemverConstraint,
			helmCreds,
		)
		if err != nil {
			if sub.Name == "" {
				return nil, fmt.Errorf(
					"error searching for latest version of chart in repository %q: %w",
					sub.RepoURL,
					err,
				)
			}
			return nil, fmt.Errorf(
				"error searching for latest version of chart %q in repository %q: %w",
				sub.Name,
				sub.RepoURL,
				err,
			)
		}

		if vers == "" {
			logger.Error("found no suitable chart version")
			if sub.Name == "" {
				return nil, fmt.Errorf(
					"found no suitable version of chart in repository %q",
					sub.RepoURL,
				)
			}
			return nil, fmt.Errorf(
				"found no suitable version of chart %q in repository %q",
				sub.Name,
				sub.RepoURL,
			)
		}
		logger.WithField("version", vers).
			Debug("found latest suitable chart version")

		charts = append(
			charts,
			kargoapi.Chart{
				RepoURL: sub.RepoURL,
				Name:    sub.Name,
				Version: vers,
			},
		)
	}

	return charts, nil
}
