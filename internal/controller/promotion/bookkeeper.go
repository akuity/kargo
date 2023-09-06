package promotion

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/akuity/bookkeeper"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
)

// bookkeeperMechanism is an implementation of the Mechanism interface that uses
// Bookkeeper to update configuration in a Git repository.
type bookkeeperMechanism struct {
	// Overridable behaviors:
	doSingleUpdateFn func(
		ctx context.Context,
		namespace string,
		update kargoapi.GitRepoUpdate,
		newFreight kargoapi.Freight,
		images []string,
	) (kargoapi.Freight, error)
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
		bookkeeper.RenderRequest,
	) (bookkeeper.RenderResponse, error)
}

// newBookkeeperMechanism returns an implementation of the Mechanism interface
// that uses Bookkeeper to update configuration in a Git repository.
func newBookkeeperMechanism(
	credentialsDB credentials.Database,
	bookkeeperService bookkeeper.Service,
) Mechanism {
	b := &bookkeeperMechanism{}
	b.doSingleUpdateFn = b.doSingleUpdate
	b.getReadRefFn = getReadRef
	b.getCredentialsFn = credentialsDB.Get
	b.renderManifestsFn = bookkeeperService.RenderManifests
	return b
}

// GetName implements the Mechanism interface.
func (*bookkeeperMechanism) GetName() string {
	return "Bookkeeper promotion mechanisms"
}

// Promote implements the Mechanism interface.
func (b *bookkeeperMechanism) Promote(
	ctx context.Context,
	stage *kargoapi.Stage,
	newFreight kargoapi.Freight,
) (kargoapi.Freight, error) {
	var updates []kargoapi.GitRepoUpdate
	for _, update := range stage.Spec.PromotionMechanisms.GitRepoUpdates {
		if update.Bookkeeper != nil {
			updates = append(updates, update)
		}
	}

	if len(updates) == 0 {
		return newFreight, nil
	}

	newFreight = *newFreight.DeepCopy()

	logger := logging.LoggerFromContext(ctx)
	logger.Debug("executing Bookkeeper-based promotion mechanisms")

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

	logger.Debug("done executing Bookkeeper-based promotion mechanisms")

	return newFreight, nil
}

// doSingleUpdateFn updates configuration in a single Git repository using
// Bookkeeper.
func (b *bookkeeperMechanism) doSingleUpdate(
	ctx context.Context,
	namespace string,
	update kargoapi.GitRepoUpdate,
	newFreight kargoapi.Freight,
	images []string,
) (kargoapi.Freight, error) {
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
	repoCreds := bookkeeper.RepoCredentials{}
	if ok {
		repoCreds.Username = creds.Username
		repoCreds.Password = creds.Password
		repoCreds.SSHPrivateKey = creds.SSHPrivateKey
		logger.Debug("obtained credentials for git repo")
	} else {
		logger.Debug("found no credentials for git repo")
	}

	req := bookkeeper.RenderRequest{
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
			"error rendering manifests for git repo %q via Bookkeeper",
			update.RepoURL,
		)
	}
	switch res.ActionTaken {
	case bookkeeper.ActionTakenPushedDirectly:
		logger.WithField("commit", res.CommitID).
			Debug("pushed new commit to repo via Bookkeeper")
		if commitIndex > -1 {
			newFreight.Commits[commitIndex].HealthCheckCommit = res.CommitID
		}
	case bookkeeper.ActionTakenNone:
		logger.Debug("Bookkeeper made no changes to repo")
		if commitIndex > -1 {
			newFreight.Commits[commitIndex].HealthCheckCommit = res.CommitID
		}
	default:
		// TODO: Not sure yet how to handle PRs.
	}

	return newFreight, nil
}
