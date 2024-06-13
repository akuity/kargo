package promotion

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	render "github.com/akuity/kargo/internal/kargo-render"
)

// newKargoRenderMechanism returns a gitMechanism that only only selects and
// performs updates that involve Kargo Render.
func newKargoRenderMechanism(
	cl client.Client,
	credentialsDB credentials.Database,
) Mechanism {
	return newGitMechanism(
		"Kargo Render promotion mechanism",
		cl,
		credentialsDB,
		selectKargoRenderUpdates,
		(&renderer{
			client:            cl,
			renderManifestsFn: render.RenderManifests,
		}).apply,
	)
}

// selectKargoRenderUpdates returns a subset of the given updates that involve
// Kargo Render.
func selectKargoRenderUpdates(updates []kargoapi.GitRepoUpdate) []*kargoapi.GitRepoUpdate {
	selectedUpdates := make([]*kargoapi.GitRepoUpdate, 0, len(updates))
	for i := range updates {
		update := &updates[i]
		if update.Render != nil {
			selectedUpdates = append(selectedUpdates, update)
		}
	}
	return selectedUpdates
}

// renderer is a helper struct whose sole purpose is to close over several
// other functions that are used in the implementation of the apply() function.
type renderer struct {
	client            client.Client
	renderManifestsFn func(req render.Request) error
}

// apply uses Kargo Render to carry out the provided update in the specified
// working directory.
func (r *renderer) apply(
	ctx context.Context,
	stage *kargoapi.Stage,
	update *kargoapi.GitRepoUpdate,
	newFreight []kargoapi.FreightReference,
	sourceCommit string,
	_ string,
	workingDir string,
	repoCreds git.RepoCredentials,
) ([]string, error) {
	images := map[string]struct{}{}
	if len(update.Render.Images) == 0 {
		// When no explicit image updates are specified, we will pass all images
		// from the Freight in <url>:<tag> format.
		desiredOrigin := freight.GetDesiredOrigin(ctx, stage, update.Render)
		for _, f := range newFreight {
			for _, image := range f.Images {
				// We actually need to "find" the image, because that will take origins
				// into account.
				foundImage, err := freight.FindImage(
					ctx,
					r.client,
					stage,
					desiredOrigin,
					newFreight,
					image.RepoURL,
				)
				if err != nil {
					return nil,
						fmt.Errorf("error finding image from repo %q: %w", image.RepoURL, err)
				}
				if foundImage != nil {
					images[fmt.Sprintf("%s:%s", image.RepoURL, image.Tag)] = struct{}{}
				}
			}
		}
	} else {
		// When explicit image updates are specified, we will only pass images with
		// a corresponding update.
		for i := range update.Render.Images {
			imageUpdate := &update.Render.Images[i]
			desiredOrigin := freight.GetDesiredOrigin(ctx, stage, imageUpdate)
			image, err := freight.FindImage(
				ctx,
				r.client,
				stage,
				desiredOrigin,
				newFreight,
				imageUpdate.Image,
			)
			if err != nil {
				return nil,
					fmt.Errorf("error finding image from repo %q: %w", imageUpdate.Image, err)
			}
			if image != nil {
				if imageUpdate.UseDigest {
					images[fmt.Sprintf("%s@%s", imageUpdate.Image, image.Digest)] = struct{}{}
				} else {
					images[fmt.Sprintf("%s:%s", imageUpdate.Image, image.Tag)] = struct{}{}
				}
			}
		}
	}

	imageList := make([]string, 0, len(images))
	for image := range images {
		imageList = append(imageList, image)
	}
	slices.Sort(imageList)

	tempDir, err := os.MkdirTemp("", tmpPrefix)
	if err != nil {
		return nil, fmt.Errorf("error creating temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	writeDir := filepath.Join(tempDir, "rendered-manifests")

	req := render.Request{
		TargetBranch: update.WriteBranch,
		Images:       imageList,
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
	for _, image := range imageList {
		changeSummary = append(
			changeSummary,
			fmt.Sprintf("updated manifests to use image %s", image),
		)
	}

	return changeSummary, nil
}
