package git

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func GetLatestCommitID(
	repoURL string,
	branch string,
	creds *Credentials,
) (string, error) {
	repo, err := Clone(repoURL, creds)
	if err != nil {
		return "", errors.Wrapf(err, "error cloning git repo %q", repoURL)

	}
	if branch != "" {
		if err = repo.Checkout(branch); err != nil {
			return "", errors.Wrapf(
				err,
				"error checking out branch %q from git repo",
				repoURL,
			)
		}
	}
	commit, err := repo.LastCommitID()
	if branch != "" {
		return commit, errors.Wrapf(
			err,
			"error determining last commit ID from branch %q of git repo %q",
			branch,
			repoURL,
		)
	}
	return commit, errors.Wrapf(
		err,
		"error determining last commit ID from default branch of git repo %q",
		repoURL,
	)
}

func ApplyUpdate(
	repoURL string,
	readRef string,
	writeBranch string,
	creds *Credentials,
	updateFn func(homeDir, workingDir string) (string, error),
) (string, error) {
	repo, err := Clone(repoURL, creds)
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
		tempDir, err := os.MkdirTemp("", "")
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

		if branchExists, err := repo.RemoteBranchExists(writeBranch); err != nil {
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
