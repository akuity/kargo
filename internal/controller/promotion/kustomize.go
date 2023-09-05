package promotion

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/kustomize"
)

// newKustomizeMechanism returns a gitMechanism that only only selects and
// performs updates that involve Kustomize.
func newKustomizeMechanism(
	credentialsDB credentials.Database,
) Mechanism {
	return newGitMechanism(
		"Kustomize promotion mechanism",
		credentialsDB,
		selectKustomizeUpdates,
		(&kustomizer{
			setImageFn: kustomize.SetImage,
		}).apply,
	)
}

// selectKustomizeUpdates returns a subset of the given updates that involve
// Kustomize.
func selectKustomizeUpdates(updates []api.GitRepoUpdate) []api.GitRepoUpdate {
	var selectedUpdates []api.GitRepoUpdate
	for _, update := range updates {
		if update.Kustomize != nil {
			selectedUpdates = append(selectedUpdates, update)
		}
	}
	return selectedUpdates
}

// kustomizer is a helper struct whose sole purpose is to close over several
// other functions that are used in the implementation of the apply() function.
type kustomizer struct {
	setImageFn func(dir, image, tag string) error
}

// apply uses Kustomize to carry out the provided update in the specified
// working directory.
func (k *kustomizer) apply(
	update api.GitRepoUpdate,
	newFreight api.Freight,
	_ string,
	workingDir string,
) ([]string, error) {
	var changeSummary []string
	for _, imgUpdate := range update.Kustomize.Images {
		var tag string
		for _, img := range newFreight.Images {
			if img.RepoURL == imgUpdate.Image {
				tag = img.Tag
				break
			}
		}
		if tag == "" {
			// TODO: Warn?
			continue
		}
		dir := filepath.Join(workingDir, imgUpdate.Path)
		if err := k.setImageFn(dir, imgUpdate.Image, tag); err != nil {
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
