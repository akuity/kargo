package environments

import (
	"context"

	"github.com/pkg/errors"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/images"
	"github.com/akuity/kargo/internal/logging"
)

func (r *reconciler) getLatestImages(
	ctx context.Context,
	namespace string,
	subs []api.ImageSubscription,
) ([]api.Image, error) {
	imgs := make([]api.Image, len(subs))
	for i, sub := range subs {

		logger := logging.LoggerFromContext(ctx).WithField("repo", sub.RepoURL)

		creds, ok, err :=
			r.credentialsDB.Get(ctx, namespace, credentials.TypeImage, sub.RepoURL)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error obtaining credentials for image repo %q",
				sub.RepoURL,
			)
		}
		var regCreds *images.Credentials
		if ok {
			regCreds = &images.Credentials{
				Username: creds.Username,
				Password: creds.Password,
			}
			logger.Debug("obtained credentials for image repo")
		} else {
			logger.Debug("found no credentials for image repo")
		}

		tag, err := r.getLatestTagFn(
			ctx,
			sub.RepoURL,
			images.ImageUpdateStrategy(sub.UpdateStrategy),
			sub.SemverConstraint,
			sub.AllowTags,
			sub.IgnoreTags,
			sub.Platform,
			regCreds,
		)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error getting latest suitable tag for image %q",
				sub.RepoURL,
			)
		}
		imgs[i] = api.Image{
			RepoURL: sub.RepoURL,
			Tag:     tag,
		}
		logger.WithField("tag", tag).
			Debug("found latest suitable image tag")
	}
	return imgs, nil
}
