package argocd

import (
	"context"

	"github.com/pkg/errors"

	"github.com/akuityio/kargo/internal/git"
	"github.com/akuityio/kargo/internal/helm"
)

func GetGitRepoCredentials(
	ctx context.Context,
	argoDB DB,
	repoURL string,
) (*git.RepoCredentials, error) {
	const repoTypeGit = "git"

	repo, err := argoDB.GetRepository(ctx, repoURL)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error getting Repository (Secret) for repo %q",
			repoURL,
		)
	}
	if repo != nil &&
		(repo.Type == repoTypeGit || repo.Type == "") &&
		(repo.Password != "" || repo.SSHPrivateKey != "") {
		return &git.RepoCredentials{
			Username:      repo.Username,
			Password:      repo.Password,
			SSHPrivateKey: repo.SSHPrivateKey,
		}, nil
	}

	repoCreds, err := argoDB.GetRepositoryCredentials(ctx, repoURL)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error getting Repository Credentials (Secret) for repo %q",
			repoURL,
		)
	}
	if repoCreds != nil &&
		(repoCreds.Type == repoTypeGit || repoCreds.Type == "") &&
		(repoCreds.Password != "" || repoCreds.SSHPrivateKey != "") {
		return &git.RepoCredentials{
			Username:      repoCreds.Username,
			Password:      repoCreds.Password,
			SSHPrivateKey: repoCreds.SSHPrivateKey,
		}, nil
	}

	return nil, nil
}

func GetChartRegistryCredentials(
	ctx context.Context,
	argoDB DB,
	registryURL string,
) (*helm.RegistryCredentials, error) {
	const repoTypeHelm = "helm"

	// NB: Argo CD Application resources typically reference git repositories.
	// They can also reference Helm charts, and in such cases, use the same
	// repository field references a REGISTRY URL. So it seems a bit awkward here,
	// but we're correct to call e.argoDB.GetRepository to look for REGISTRY
	// credentials.
	repo, err := argoDB.GetRepository(ctx, registryURL)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error getting Argo CD Repository (Secret) for Helm chart registry %q",
			registryURL,
		)
	}
	if repo != nil && repo.Type == repoTypeHelm && repo.Password != "" {
		return &helm.RegistryCredentials{
			Username: repo.Username,
			Password: repo.Password,
		}, nil
	}

	repoCreds, err := argoDB.GetRepositoryCredentials(ctx, registryURL)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error getting Argo CD Repository Credentials (Secret) for Helm chart "+
				"registry %q",
			registryURL,
		)
	}
	if repoCreds != nil &&
		repoCreds.Type == repoTypeHelm &&
		repoCreds.Password != "" {
		return &helm.RegistryCredentials{
			Username: repoCreds.Username,
			Password: repoCreds.Password,
		}, nil
	}

	return nil, nil
}
