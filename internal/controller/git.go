package controller

import (
	"context"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/db"
	"github.com/pkg/errors"

	"github.com/akuityio/k8sta/internal/git"
)

// TODO: Implement this
func getRepoCredentials(
	ctx context.Context,
	repoURL string,
	argoDB db.ArgoDB,
) (git.RepoCredentials, error) {
	const repoTypeGit = "git"

	creds := git.RepoCredentials{}

	// NB: This next call returns an empty Repository if no such Repository is
	// found, so instead of continuing to look for credentials if no Repository is
	// found, what we'll do is continue looking for credentials if the Repository
	// we get back doesn't have anything we can use, i.e. no SSH private key or
	// password.
	repo, err := argoDB.GetRepository(ctx, repoURL)
	if err != nil {
		return creds, errors.Wrapf(
			err,
			"error getting Repository (Secret) for repo %q",
			repoURL,
		)
	}
	if repo.Type == repoTypeGit || repo.Type == "" {
		creds.SSHPrivateKey = repo.SSHPrivateKey
		creds.Username = repo.Username
		creds.Password = repo.Password
	}
	if creds.SSHPrivateKey == "" && creds.Password == "" {
		// We didn't find any creds yet, so keep looking
		var repoCreds *argocd.RepoCreds
		repoCreds, err = argoDB.GetRepositoryCredentials(ctx, repoURL)
		if err != nil {
			return creds, errors.Wrapf(
				err,
				"error getting Repository Credentials (Secret) for repo %q",
				repoURL,
			)
		}
		if repoCreds.Type == repoTypeGit || repoCreds.Type == "" {
			creds.SSHPrivateKey = repo.SSHPrivateKey
			creds.Username = repo.Username
			creds.Password = repo.Password
		}
	}

	// We didn't find any creds, so we're done. We need creds.
	if creds.SSHPrivateKey == "" && creds.Password == "" {
		return creds, errors.Errorf(
			"could not find any credentials for repo %q",
			repoURL,
		)
	}

	return creds, nil
}
