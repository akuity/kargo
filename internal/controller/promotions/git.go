package promotions

import (
	"context"

	"github.com/pkg/errors"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/logging"
)

func (r *reconciler) applyGitRepoUpdate(
	ctx context.Context,
	namespace string,
	newState api.EnvironmentState,
	update api.GitRepoUpdate,
) (api.EnvironmentState, error) {
	newState = *newState.DeepCopy()

	logger := logging.LoggerFromContext(ctx).WithField("repo", update.RepoURL)

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

	creds, ok, err :=
		r.credentialsDB.Get(ctx, namespace, credentials.TypeGit, update.RepoURL)
	if err != nil {
		return newState, errors.Wrapf(
			err,
			"error obtaining credentials for git repo %q",
			update.RepoURL,
		)
	}
	var repoCreds *git.Credentials
	if ok {
		repoCreds = &git.Credentials{
			Username:      creds.Username,
			Password:      creds.Password,
			SSHPrivateKey: creds.SSHPrivateKey,
		}
		logger.Debug("obtained credentials for git repo")
	} else {
		logger.Debug("found no credentials for git repo")
	}

	commitID, err := r.gitApplyUpdateFn(
		update.RepoURL,
		readRef,
		update.WriteBranch,
		repoCreds,
		func(homeDir, workingDir string) (string, error) {
			if update.Kustomize != nil {
				if err = r.applyKustomize(
					newState,
					*update.Kustomize,
					workingDir,
				); err != nil {
					return "", errors.Wrapf(
						err,
						"error updating branch %q in git repository %q via Kustomize",
						update.WriteBranch,
						update.RepoURL,
					)
				}
			}

			if update.Helm != nil {
				if err = r.applyHelm(
					newState,
					*update.Helm,
					homeDir,
					workingDir,
				); err != nil {
					return "", errors.Wrapf(
						err,
						"error updating branch %q in git repository %q via Helm",
						update.WriteBranch,
						update.RepoURL,
					)
				}
			}

			// TODO: This is an awful commit message! Fix it!
			return "kargo made some changes!", nil
		},
	)
	if err != nil {
		return newState, err
	}

	if commitIndex > -1 {
		newState.Commits[commitIndex].HealthCheckCommit = commitID
	}

	return newState, nil
}
