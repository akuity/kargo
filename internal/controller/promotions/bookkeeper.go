package promotions

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/akuity/bookkeeper"
	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
)

func (r *reconciler) applyBookkeeperUpdate(
	ctx context.Context,
	namespace string,
	newState api.EnvironmentState,
	update api.GitRepoUpdate,
) (api.EnvironmentState, error) {
	if update.Bookkeeper == nil {
		return newState, nil
	}

	var readRef string
	commitIndex := -1
	for i, commit := range newState.Commits {
		if commit.RepoURL == update.RepoURL {
			if update.WriteBranch == commit.Branch {
				return newState, errors.Errorf(
					"invalid update specified; cannot write to branch %q of repo %q "+
						"because it will form a subscription loop",
					update.RepoURL,
					update.WriteBranch,
				)
			}
			commitIndex = i
			readRef = commit.ID
			break
		}
	}
	if readRef == "" {
		readRef = update.ReadBranch
	}

	images := make([]string, len(newState.Images))
	for i, image := range newState.Images {
		images[i] = fmt.Sprintf("%s:%s", image.RepoURL, image.Tag)
	}

	logger := logging.LoggerFromContext(ctx).WithField("repo", update.RepoURL)

	creds, ok, err :=
		r.credentialsDB.Get(ctx, namespace, credentials.TypeGit, update.RepoURL)
	if err != nil {
		return newState, errors.Wrapf(
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
		Commit:       readRef,
		Images:       images,
		TargetBranch: update.WriteBranch,
	}
	res, err := r.bookkeeperService.RenderManifests(ctx, req)
	if err != nil {
		return newState,
			errors.Wrap(err, "error rendering manifests via Bookkeeper")
	}

	switch res.ActionTaken {
	case bookkeeper.ActionTakenPushedDirectly:
		logger.WithField("commit", res.CommitID).
			Debug("pushed new commit to repo via Bookkeeper")
		if commitIndex > -1 {
			newState.Commits[commitIndex].HealthCheckCommit = res.CommitID
		}
	case bookkeeper.ActionTakenNone:
		logger.Debug("Bookkeeper made no changes to repo")
		if commitIndex > -1 {
			newState.Commits[commitIndex].HealthCheckCommit = res.CommitID
		}
	default:
		// TODO: Not sure yet how to handle PRs.
	}

	return newState, nil
}
