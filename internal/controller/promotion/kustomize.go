package promotion

import (
	"context"
	"fmt"
	"path/filepath"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/kustomize"
)

// newKustomizeMechanism returns a gitMechanism that only only selects and
// performs updates that involve Kustomize.
func newKustomizeMechanism(
	cl client.Client,
	credentialsDB credentials.Database,
) Mechanism {
	return newGitMechanism(
		"Kustomize promotion mechanism",
		cl,
		credentialsDB,
		selectKustomizeUpdates,
		(&kustomizer{
			client:      cl,
			findImageFn: freight.FindImage,
			setImageFn:  kustomize.SetImage,
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
	client      client.Client
	findImageFn func(
		ctx context.Context,
		cl client.Client,
		stage *kargoapi.Stage,
		desiredOrigin *kargoapi.FreightOrigin,
		freight []kargoapi.FreightReference,
		repoURL string,
	) (*kargoapi.Image, error)
	setImageFn func(dir, fqImageRef string) error
}

// apply uses Kustomize to carry out the provided update in the specified
// working directory.
func (k *kustomizer) apply(
	ctx context.Context,
	stage *kargoapi.Stage,
	update *kargoapi.GitRepoUpdate,
	newFreight []kargoapi.FreightReference,
	_ string, // TODO: sourceCommit would be a nice addition to the commit message
	_ string,
	workingDir string,
	_ git.RepoCredentials,
) ([]string, error) {
	changeSummary := make([]string, 0, len(update.Kustomize.Images))
	for i := range update.Kustomize.Images {
		imgUpdate := &update.Kustomize.Images[i]
		desiredOrigin := freight.GetDesiredOrigin(stage, imgUpdate)
		image, err := k.findImageFn(ctx, k.client, stage, desiredOrigin, newFreight, imgUpdate.Image)
		if err != nil {
			return nil,
				fmt.Errorf("error finding image %q from Freight: %w", imgUpdate.Image, err)
		}
		if image == nil {
			// TODO: Warn?
			continue
		}
		var fqImageRef string // Fully-qualified image reference
		if imgUpdate.UseDigest {
			fqImageRef = fmt.Sprintf("%s@%s", image.RepoURL, image.Digest)
		} else {
			fqImageRef = fmt.Sprintf("%s:%s", image.RepoURL, image.Tag)
		}
		dir := filepath.Join(workingDir, imgUpdate.Path)
		if err := k.setImageFn(dir, fqImageRef); err != nil {
			return nil, fmt.Errorf(
				"error updating image %q to %q using Kustomize: %w",
				imgUpdate.Image,
				fqImageRef,
				err,
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
