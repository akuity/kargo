package promotion

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	render "github.com/akuity/kargo-render"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
)

// kargoRenderMechanism is an implementation of the Mechanism interface that
// uses Kargo Render to update configuration in a Git repository.
type kargoRenderMechanism struct {
	// Overridable behaviors:
	doSingleUpdateFn func(
		ctx context.Context,
		namespace string,
		update kargoapi.GitRepoUpdate,
		newFreight kargoapi.SimpleFreight,
		images []string,
	) (kargoapi.SimpleFreight, error)
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
	renderManifestsFn func(
		context.Context,
		render.Request,
	) (render.Response, error)
}

// newKargoRenderMechanism returns an implementation of the Mechanism interface
// that uses Kargo Render to update configuration in a Git repository.
func newKargoRenderMechanism(
	credentialsDB credentials.Database,
	renderService render.Service,
) Mechanism {
	b := &kargoRenderMechanism{}
	b.doSingleUpdateFn = b.doSingleUpdate
	b.getReadRefFn = getReadRef
	b.getCredentialsFn = credentialsDB.Get
	b.renderManifestsFn = renderService.RenderManifests
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
	newFreight kargoapi.SimpleFreight,
) (kargoapi.SimpleFreight, error) {
	updates := make([]kargoapi.GitRepoUpdate, 0, len(stage.Spec.PromotionMechanisms.GitRepoUpdates))
	for _, update := range stage.Spec.PromotionMechanisms.GitRepoUpdates {
		if update.Render != nil {
			updates = append(updates, update)
		}
	}

	if len(updates) == 0 {
		return newFreight, nil
	}

	newFreight = *newFreight.DeepCopy()

	logger := logging.LoggerFromContext(ctx)
	logger.Debug("executing Kargo Render-based promotion mechanisms")

	images := make([]string, len(newFreight.Images))
	for i, image := range newFreight.Images {
		images[i] = fmt.Sprintf("%s:%s", image.RepoURL, image.Tag)
	}

	for _, update := range updates {
		var err error
		if newFreight, err = b.doSingleUpdateFn(
			ctx,
			stage.Namespace,
			update,
			newFreight,
			images,
		); err != nil {
			return newFreight, err
		}
	}

	logger.Debug("done executing Kargo Render-based promotion mechanisms")

	return newFreight, nil
}

// doSingleUpdateFn updates configuration in a single Git repository using
// Kargo Render.
func (b *kargoRenderMechanism) doSingleUpdate(
	ctx context.Context,
	namespace string,
	update kargoapi.GitRepoUpdate,
	newFreight kargoapi.SimpleFreight,
	images []string,
) (kargoapi.SimpleFreight, error) {
	logger := logging.LoggerFromContext(ctx).WithField("repo", update.RepoURL)

	readRef, commitIndex, err := b.getReadRefFn(update, newFreight.Commits)
	if err != nil {
		return newFreight, err
	}

	creds, ok, err := b.getCredentialsFn(
		ctx,
		namespace,
		credentials.TypeGit,
		update.RepoURL,
	)
	if err != nil {
		return newFreight, errors.Wrapf(
			err,
			"error obtaining credentials for git repo %q",
			update.RepoURL,
		)
	}
	repoCreds := render.RepoCredentials{}
	if ok {
		repoCreds.Username = creds.Username
		repoCreds.Password = creds.Password
		repoCreds.SSHPrivateKey = creds.SSHPrivateKey
		logger.Debug("obtained credentials for git repo")
	} else {
		logger.Debug("found no credentials for git repo")
	}

	req := render.Request{
		RepoURL:      update.RepoURL,
		RepoCreds:    repoCreds,
		Ref:          readRef,
		Images:       images,
		TargetBranch: update.WriteBranch,
	}

	res, err := b.renderManifestsFn(ctx, req)
	if err != nil {
		return newFreight, errors.Wrapf(
			err,
			"error rendering manifests for git repo %q via Kargo Render",
			update.RepoURL,
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
