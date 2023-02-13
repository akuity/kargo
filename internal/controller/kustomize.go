package controller

import (
	"context"
	"path/filepath"

	"github.com/pkg/errors"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/git"
	"github.com/akuityio/kargo/internal/kustomize"
)

// TODO: Add some logging to this function
// nolint: gocyclo
func (e *environmentReconciler) promoteWithKustomize(
	ctx context.Context,
	env *api.Environment,
	newState api.EnvironmentState,
) (api.EnvironmentState, error) {
	if env == nil ||
		env.Spec.PromotionMechanisms == nil ||
		env.Spec.PromotionMechanisms.Git == nil ||
		env.Spec.PromotionMechanisms.Git.Kustomize == nil ||
		len(env.Spec.PromotionMechanisms.Git.Kustomize.Images) == 0 ||
		len(newState.Images) == 0 {
		return newState, nil
	}

	if env.Spec.GitRepo == nil || env.Spec.GitRepo.URL == "" {
		return newState, errors.New(
			"cannot promote images via Kustomize because spec does not contain " +
				"git repo details",
		)
	}

	return e.promoteWithGit(
		ctx,
		env,
		newState,
		"updating images",
		func(repo git.Repo) error {
			imgUpdates := env.Spec.PromotionMechanisms.Git.Kustomize.Images
			for _, imgUpdate := range imgUpdates {
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
				dir := filepath.Join(repo.WorkingDir(), imgUpdate.Path)
				if err := kustomize.SetImage(dir, imgUpdate.Image, tag); err != nil {
					return errors.Wrapf(
						err,
						"error updating image %q to tag %q using Kustomize",
						imgUpdate.Image,
						tag,
					)
				}
			}
			return nil
		},
	)
}
