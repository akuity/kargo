package promotion

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	render "github.com/akuity/kargo/internal/kargo-render"
)

// newKargoRenderMechanism returns a gitMechanism that only only selects and
// performs updates that involve Kargo Render.
func newKargoRenderMechanism(
	credentialsDB credentials.Database,
) Mechanism {
	return newGitMechanism(
		"Kargo Render promotion mechanism",
		credentialsDB,
		selectKargoRenderUpdates,
		(&renderer{
			renderManifestsFn: render.RenderManifests,
		}).apply,
	)
}

// selectKargoRenderUpdates returns a subset of the given updates that involve
// Kargo Render.
func selectKargoRenderUpdates(updates []kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate {
	selectedUpdates := make([]kargoapi.GitRepoUpdate, 0, len(updates))
	for _, update := range updates {
		if update.Render != nil {
			selectedUpdates = append(selectedUpdates, update)
		}
	}
	return selectedUpdates
}

// renderer is a helper struct whose sole purpose is to close over several
// other functions that are used in the implementation of the apply() function.
type renderer struct {
	renderManifestsFn func(req render.Request) error
}

// apply uses Kargo Render to carry out the provided update in the specified
// working directory.
func (r *renderer) apply(
	_ context.Context,
	update kargoapi.GitRepoUpdate,
	newFreight kargoapi.FreightReference,
	_ string,
	sourceCommit string,
	_ string,
	workingDir string,
	repoCreds git.RepoCredentials,
) ([]string, error) {
	images := make([]string, 0, len(newFreight.Images))
	if len(update.Render.Images) == 0 {
		// When no explicit image updates are specified, we will pass all images
		// from the Freight in <ulr>:<tag> format.
		for _, image := range newFreight.Images {
			images = append(images, fmt.Sprintf("%s:%s", image.RepoURL, image.Tag))
		}
	} else {
		// When explicit image updates are specified, we will only pass images with
		// a corresponding update.

		// Build a map of image updates indexed by image URL. This way, as we
		// iterate over all images in the Freight, we can quickly check if there is
		// an update, and if so, whether it specifies to use a digest or a tag.
		imageUpdatesByImage :=
			make(map[string]kargoapi.KargoRenderImageUpdate, len(update.Render.Images))
		for _, imageUpdate := range update.Render.Images {
			imageUpdatesByImage[imageUpdate.Image] = imageUpdate
		}
		for _, image := range newFreight.Images {
			if imageUpdate, ok := imageUpdatesByImage[image.RepoURL]; ok {
				if imageUpdate.UseDigest {
					images = append(images, fmt.Sprintf("%s@%s", image.RepoURL, image.Digest))
				} else {
					images = append(images, fmt.Sprintf("%s:%s", image.RepoURL, image.Tag))
				}
			}
		}
	}

	slices.Sort(images)
	tempDir, err := os.MkdirTemp("", tmpPrefix)
	if err != nil {
		return nil, fmt.Errorf("error creating temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	writeDir := filepath.Join(tempDir, "rendered-manifests")

	req := render.Request{
		TargetBranch: update.WriteBranch,
		Images:       images,
		LocalInPath:  workingDir,
		LocalOutPath: writeDir,
		RepoCreds:    repoCreds,
	}

	if err = r.renderManifestsFn(req); err != nil {
		return nil, fmt.Errorf("error rendering manifests via Kargo Render: %w", err)
	}

	if err = deleteRepoContents(workingDir); err != nil {
		return nil,
			fmt.Errorf("error overwriting working directory with rendered manifests: %w", err)
	}

	if err = moveRepoContents(writeDir, workingDir); err != nil {
		return nil,
			fmt.Errorf("error overwriting working directory with rendered manifests: %w", err)
	}

	changeSummary := make([]string, 0, len(images)+1)
	changeSummary = append(
		changeSummary,
		fmt.Sprintf("rendered manifests from commit %s", sourceCommit[:7]),
	)
	for _, image := range images {
		changeSummary = append(
			changeSummary,
			fmt.Sprintf("updated manifests to use image %s", image),
		)
	}

	return changeSummary, nil
}
