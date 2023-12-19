package warehouses

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/logging"
)

func (r *reconciler) getLatestCharts(
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

		logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
			"repository": sub.Repository,
			"chart":      sub.Name,
		})

		creds, ok, err :=
			r.credentialsDB.Get(ctx, namespace, credentials.TypeHelm, sub.Repository)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error obtaining credentials for chart repository %q",
				sub.Repository,
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

		vers, err := r.getLatestChartVersionFn(
			ctx,
			sub.Repository,
			sub.Name,
			sub.SemverConstraint,
			helmCreds,
		)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error searching for latest version of chart %q in repository %q",
				sub.Name,
				sub.Repository,
			)
		}

		if vers == "" {
			logger.Error("found no suitable chart version")
			return nil, errors.Errorf(
				"found no suitable version of chart %q in repository %q",
				sub.Name,
				sub.Repository,
			)
		}
		logger.WithField("version", vers).
			Debug("found latest suitable chart version")

		charts = append(
			charts,
			kargoapi.Chart{
				Repository: sub.Repository,
				Name:       sub.Name,
				Version:    vers,
			},
		)
	}

	return charts, nil
}
