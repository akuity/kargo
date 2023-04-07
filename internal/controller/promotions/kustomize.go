package promotions

import (
	"path/filepath"

	"github.com/pkg/errors"

	api "github.com/akuity/kargo/api/v1alpha1"
)

func (r *reconciler) applyKustomize(
	newState api.EnvironmentState,
	update api.KustomizePromotionMechanism,
	repoDir string,
) error {
	for _, imgUpdate := range update.Images {
		var tag string
		for _, img := range newState.Images {
			if img.RepoURL == imgUpdate.Image {
				tag = img.Tag
				break
			}
		}
		if tag == "" {
			// TODO: Warn?
			continue
		}
		dir := filepath.Join(repoDir, imgUpdate.Path)
		if err := r.kustomizeSetImageFn(dir, imgUpdate.Image, tag); err != nil {
			return errors.Wrapf(
				err,
				"error updating image %q to tag %q using Kustomize",
				imgUpdate.Image,
				tag,
			)
		}
	}
	return nil
}
