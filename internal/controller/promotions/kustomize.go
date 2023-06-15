package promotions

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

	api "github.com/akuity/kargo/api/v1alpha1"
)

func (r *reconciler) applyKustomize(
	newState api.EnvironmentState,
	update api.KustomizePromotionMechanism,
	repoDir string,
) ([]string, error) {
	changeSummary := []string{}

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
			return nil, errors.Wrapf(
				err,
				"error updating image %q to tag %q using Kustomize",
				imgUpdate.Image,
				tag,
			)
		}
		changeSummary = append(
			changeSummary,
			fmt.Sprintf(
				"updated %s/kustomization.yaml to use image %s:%s",
				imgUpdate.Path,
				imgUpdate.Image,
				tag,
			),
		)
	}

	return changeSummary, nil
}
