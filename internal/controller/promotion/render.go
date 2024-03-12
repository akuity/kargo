package promotion

import (
	"context"
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	render "github.com/akuity/kargo/internal/kargo-render"
	"github.com/akuity/kargo/internal/logging"
)

// kargoRenderMechanism is an implementation of the Mechanism interface that
// uses Kargo Render to update configuration in a Git repository.
type kargoRenderMechanism struct {
	// Overridable behaviors:
	doSingleUpdateFn func(
		ctx context.Context,
		promo *kargoapi.Promotion,
		update kargoapi.GitRepoUpdate,
		newFreight kargoapi.FreightReference,
	) (kargoapi.FreightReference, error)
	getReadRefFn func(
		update kargoapi.GitRepoUpdate,
		commits []kargoapi.GitCommit,
	) (string, int, error)
	getCredentialsFn func(
		ctx context.Context,
		namespace string,
		credType credentials.Type,
		repo string,
	) (credentials.Credentials, bool, error)
	renderManifestsFn func(render.Request) (render.Response, error)
}

// newKargoRenderMechanism returns an implementation of the Mechanism interface
// that uses Kargo Render to update configuration in a Git repository.
func newKargoRenderMechanism(
	credentialsDB credentials.Database,
) Mechanism {
	b := &kargoRenderMechanism{}
	b.doSingleUpdateFn = b.doSingleUpdate
	b.getReadRefFn = getReadRef
	b.getCredentialsFn = credentialsDB.Get
	// TODO: KR: Refactor this
	b.renderManifestsFn = render.RenderManifests
	return b
}

// GetName implements the Mechanism interface.
func (*kargoRenderMechanism) GetName() string {
	return "Kargo Render promotion mechanisms"
}

// Promote implements the Mechanism interface.
func (b *kargoRenderMechanism) Promote(
	ctx context.Context,
	stage *kargoapi.Stage,
	promo *kargoapi.Promotion,
	newFreight kargoapi.FreightReference,
) (*kargoapi.PromotionStatus, kargoapi.FreightReference, error) {
	updates := make([]kargoapi.GitRepoUpdate, 0, len(stage.Spec.PromotionMechanisms.GitRepoUpdates))
	for _, update := range stage.Spec.PromotionMechanisms.GitRepoUpdates {
		if update.Render != nil {
			updates = append(updates, update)
		}
	}

	if len(updates) == 0 {
		return promo.Status.WithPhase(kargoapi.PromotionPhaseSucceeded), newFreight, nil
	}

	newFreight = *newFreight.DeepCopy()

	logger := logging.LoggerFromContext(ctx)
	logger.Debug("executing Kargo Render-based promotion mechanisms")

	for _, update := range updates {
		var err error
		if newFreight, err = b.doSingleUpdateFn(
			ctx,
			promo,
			update,
			newFreight,
		); err != nil {
			return nil, newFreight, err
		}
	}

	logger.Debug("done executing Kargo Render-based promotion mechanisms")

	return promo.Status.WithPhase(kargoapi.PromotionPhaseSucceeded), newFreight, nil
}

// doSingleUpdateFn updates configuration in a single Git repository using
// Kargo Render.
func (b *kargoRenderMechanism) doSingleUpdate(
	ctx context.Context,
	promo *kargoapi.Promotion,
	update kargoapi.GitRepoUpdate,
	newFreight kargoapi.FreightReference,
) (kargoapi.FreightReference, error) {
	logger := logging.LoggerFromContext(ctx).WithField("repo", update.RepoURL)

	readRef, commitIndex, err := b.getReadRefFn(update, newFreight.Commits)
	if err != nil {
		return newFreight, err
	}

	creds, ok, err := b.getCredentialsFn(
		ctx,
		promo.Namespace,
		credentials.TypeGit,
		update.RepoURL,
	)
	if err != nil {
		return newFreight, fmt.Errorf(
			"error obtaining credentials for git repo %q: %w",
			update.RepoURL,
			err,
		)
	}
	repoCreds := git.RepoCredentials{}
	if ok {
		repoCreds.Username = creds.Username
		repoCreds.Password = creds.Password
		repoCreds.SSHPrivateKey = creds.SSHPrivateKey
		logger.Debug("obtained credentials for git repo")
	} else {
		logger.Debug("found no credentials for git repo")
	}

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

	req := render.Request{
		RepoURL:      update.RepoURL,
		RepoCreds:    repoCreds,
		Ref:          readRef,
		Images:       images,
		TargetBranch: update.WriteBranch,
	}

	res, err := b.renderManifestsFn(req)
	if err != nil {
		return newFreight, fmt.Errorf(
			"error rendering manifests for git repo %q via Kargo Render: %w",
			update.RepoURL,
			err,
		)
	}
	switch res.ActionTaken {
	case render.ActionTakenPushedDirectly:
		logger.WithField("commit", res.CommitID).
			Debug("pushed new commit to repo via Kargo Render")
		if commitIndex > -1 {
			newFreight.Commits[commitIndex].HealthCheckCommit = res.CommitID
		}
	case render.ActionTakenNone:
		logger.Debug("Kargo Render made no changes to repo")
		if commitIndex > -1 {
			newFreight.Commits[commitIndex].HealthCheckCommit = res.CommitID
		}
	default:
		// TODO: Not sure yet how to handle PRs.
	}

	return newFreight, nil
}
