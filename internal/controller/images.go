package controller

import (
	"context"

	"github.com/pkg/errors"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/images"
)

func (e *environmentReconciler) getLatestImages(
	ctx context.Context,
	namespace string,
	subs []api.ImageSubscription,
) ([]api.Image, error) {
	imgs := make([]api.Image, len(subs))
	for i, sub := range subs {
		creds, ok, err :=
			e.credentialsDB.get(ctx, namespace, credentialsTypeImage, sub.RepoURL)
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
		}

		tag, err := e.getLatestTagFn(
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
	}
	return imgs, nil
}
