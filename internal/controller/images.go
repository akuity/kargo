package controller

import (
	"context"

	"github.com/pkg/errors"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/images"
)

func (e *environmentReconciler) getLatestImages(
	ctx context.Context,
	subs []api.ImageSubscription,
) ([]api.Image, error) {
	imgs := make([]api.Image, len(subs))
	for i, sub := range subs {
		tag, err := e.getLatestTagFn(
			ctx,
			e.kubeClient,
			sub.RepoURL,
			images.ImageUpdateStrategy(sub.UpdateStrategy),
			sub.SemverConstraint,
			sub.AllowTags,
			sub.IgnoreTags,
			sub.Platform,
			sub.PullSecret,
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
