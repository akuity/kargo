package promotion

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
func selectKustomizeUpdates(updates []kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate {
	selectedUpdates := make([]kargoapi.GitRepoUpdate, 0, len(updates))
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
	setImageFn func(dir, fqImageRef string) error
}

// apply uses Kustomize to carry out the provided update in the specified
// working directory.
func (k *kustomizer) apply(
	update kargoapi.GitRepoUpdate,
	newFreight kargoapi.FreightReference,
	_ string,
	workingDir string,
) ([]string, error) {
	changeSummary := make([]string, 0, len(update.Kustomize.Images))
	for _, imgUpdate := range update.Kustomize.Images {
		var fqImageRef string // Fully-qualified image reference
		for _, img := range newFreight.Images {
			if img.RepoURL == imgUpdate.Image {
				if imgUpdate.UseDigest {
					fqImageRef = fmt.Sprintf("%s@%s", img.RepoURL, img.Digest)
				} else {
					fqImageRef = fmt.Sprintf("%s:%s", img.RepoURL, img.Tag)
				}
				break
			}
		}
		if fqImageRef == "" {
			// TODO: Warn?
			continue
		}
		dir := filepath.Join(workingDir, imgUpdate.Path)
		if err := k.setImageFn(dir, fqImageRef); err != nil {
			return nil, errors.Wrapf(
				err,
				"error updating image %q to %q using Kustomize",
				imgUpdate.Image,
				fqImageRef,
			)
		}
		changeSummary = append(
			changeSummary,
			fmt.Sprintf(
				"updated %s/kustomization.yaml to use image %s",
				imgUpdate.Path,
				fqImageRef,
			),
		)
	}
	return changeSummary, nil
}
