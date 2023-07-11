package promotions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/akuity/bookkeeper/pkg/git"
	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
)

func (r *reconciler) applyGitRepoUpdate(
	ctx context.Context,
	namespace string,
	newState api.StageState,
	update api.GitRepoUpdate,
) (api.StageState, error) {
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
	var repoCreds *git.RepoCredentials
	if ok {
		repoCreds = &git.RepoCredentials{
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
			changeSummary := []string{}

			if update.Kustomize != nil {
				var newChanges []string
				newChanges, err = r.applyKustomize(
					newState,
					*update.Kustomize,
					workingDir,
				)
				if err != nil {
					return "", errors.Wrapf(
						err,
						"error updating branch %q in git repository %q via Kustomize",
						update.WriteBranch,
						update.RepoURL,
					)
				}
				changeSummary = append(changeSummary, newChanges...)
			}

			if update.Helm != nil {
				var newChanges []string
				newChanges, err = r.applyHelm(
					newState,
					*update.Helm,
					homeDir,
					workingDir,
				)
				if err != nil {
					return "", errors.Wrapf(
						err,
						"error updating branch %q in git repository %q via Helm",
						update.WriteBranch,
						update.RepoURL,
					)
				}
				changeSummary = append(changeSummary, newChanges...)
			}

			return buildCommitMessage(changeSummary), nil
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

func gitApplyUpdate(
	repoURL string,
	readRef string,
	writeBranch string,
	creds *git.RepoCredentials,
	updateFn func(homeDir, workingDir string) (string, error),
) (string, error) {
	if creds == nil {
		creds = &git.RepoCredentials{}
	}
	repo, err := git.Clone(repoURL, *creds)
	if err != nil {
		return "", errors.Wrapf(err, "error cloning git repo %q", repoURL)
	}
	defer repo.Close()

	// If readRef is non-empty, check out the specified commit or branch,
	// otherwise just move using the repository's default branch as the source.
	if readRef != "" {
		if err = repo.Checkout(readRef); err != nil {
			return "", errors.Wrapf(
				err,
				"error checking out %q from git repo",
				readRef,
			)
		}
	}

	commitMsg, err := updateFn(repo.HomeDir(), repo.WorkingDir())
	if err != nil {
		return "", err
	}

	// Sometimes we don't write to the same branch we read from...
	if readRef != writeBranch {
		var tempDir string
		tempDir, err = os.MkdirTemp("", "")
		if err != nil {
			return "", errors.Wrap(
				err,
				"error creating temp directory for pending changes",
			)
		}
		defer os.RemoveAll(tempDir)

		if err = moveRepoContents(repo.WorkingDir(), tempDir); err != nil {
			return "", errors.Wrap(
				err,
				"error moving repository working tree to temporary location",
			)
		}

		if err = repo.ResetHard(); err != nil {
			return "", errors.Wrap(err, "error resetting repository working tree")
		}

		var branchExists bool
		if branchExists, err = repo.RemoteBranchExists(writeBranch); err != nil {
			return "", errors.Wrapf(
				err,
				"error checking for existence of branch %q in remote repo %q",
				writeBranch,
				repoURL,
			)
		} else if !branchExists {
			if err = repo.CreateOrphanedBranch(writeBranch); err != nil {
				return "", errors.Wrapf(
					err,
					"error creating branch %q in repo %q",
					writeBranch,
					repoURL,
				)
			}
		} else {
			if err = repo.Checkout(writeBranch); err != nil {
				return "", errors.Wrapf(
					err,
					"error checking out branch %q from git repo %q",
					writeBranch,
					repoURL,
				)
			}
		}

		if err = deleteRepoContents(repo.WorkingDir()); err != nil {
			return "",
				errors.Wrap(err, "error clearing contents from repository working tree")
		}

		if err = moveRepoContents(tempDir, repo.WorkingDir()); err != nil {
			return "", errors.Wrap(
				err,
				"error restoring repository working tree from temporary location",
			)
		}
	}

	hasDiffs, err := repo.HasDiffs()
	if err != nil {
		return "",
			errors.Wrapf(err, "error checking for diffs in git repo %q", repoURL)
	}

	if hasDiffs {
		if err = repo.AddAllAndCommit(commitMsg); err != nil {
			return "",
				errors.Wrapf(err, "error committing updates to git repo %q", repoURL)
		}
		if err = repo.Push(); err != nil {
			return "",
				errors.Wrapf(err, "error pushing updates to git repo %q", repoURL)
		}
	}

	commitID, err := repo.LastCommitID()
	if err != nil {
		return "", errors.Wrapf(
			err,
			"error getting last commit ID from git repo %q",
			repoURL,
		)
	}

	return commitID, nil
}

func moveRepoContents(srcDir, destDir string) error {
	dirEntries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, dirEntry := range dirEntries {
		if dirEntry.Name() == ".git" {
			continue
		}
		srcPath := filepath.Join(srcDir, dirEntry.Name())
		destPath := filepath.Join(destDir, dirEntry.Name())
		if err = os.Rename(srcPath, destPath); err != nil {
			return err
		}
	}
	return nil
}

func deleteRepoContents(dir string) error {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, dirEntry := range dirEntries {
		if dirEntry.Name() == ".git" {
			continue
		}
		if err = os.RemoveAll(filepath.Join(dir, dirEntry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func buildCommitMessage(changeSummary []string) string {
	if len(changeSummary) == 0 { // This shouldn't really happen
		return "Kargo applied some changes"
	}
	if len(changeSummary) == 1 {
		return changeSummary[0]
	}
	msg := "Kargo applied multiple changes\n\nIncluding:\n"
	for _, change := range changeSummary {
		msg = fmt.Sprintf("%s\n  * %s", msg, change)
	}
	return msg
}
