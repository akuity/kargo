package controller

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/akuityio/bookkeeper"
	api "github.com/akuityio/kargo/api/v1alpha1"
)

func (e *environmentReconciler) applyBookkeeperUpdate(
	ctx context.Context,
	namespace string,
	newState api.EnvironmentState,
	update api.GitRepoUpdate,
) (api.EnvironmentState, error) {
	if update.Bookkeeper == nil {
		return newState, nil
	}

	if update.Branch == "" {
		return newState, errors.Errorf(
			"cannot update repo %q using Bookkeeper because no target branch "+
				"is specified",
			update.RepoURL,
		)
	}

	var commitID string
	var commitIndex int
	var commit api.GitCommit
	for commitIndex, commit = range newState.Commits {
		if commit.RepoURL == update.RepoURL {
			commitID = commit.ID
			break
		}
	}
	if commitID == "" {
		return newState, errors.Errorf(
			"cannot update repo %q using Bookkeeper because the environment does "+
				"not subscribe to repo %q",
			update.RepoURL,
			update.RepoURL,
		)
	}

	images := make([]string, len(newState.Images))
	for i, image := range newState.Images {
		images[i] = fmt.Sprintf("%s:%s", image.RepoURL, image.Tag)
	}

	creds, ok, err :=
		e.credentialsDB.get(ctx, namespace, credentialsTypeGit, update.RepoURL)
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
	}

	req := bookkeeper.RenderRequest{
		RepoURL:      update.RepoURL,
		RepoCreds:    repoCreds,
		Commit:       commitID,
		Images:       images,
		TargetBranch: update.Branch,
	}
	res, err := e.bookkeeperService.RenderManifests(ctx, req)
	if err != nil {
		return newState,
			errors.Wrap(err, "error rendering manifests via Bookkeeper")
	}

	if res.ActionTaken == bookkeeper.ActionTakenPushedDirectly ||
		res.ActionTaken == bookkeeper.ActionTakenNone {
		newState.Commits[commitIndex].HealthCheckCommit = res.CommitID
	} // TODO: Not sure yet how to handle PRs.

	return newState, nil
}
