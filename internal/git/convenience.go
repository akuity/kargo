package git

import (
	"github.com/pkg/errors"
)

func GetLatestCommitID(
	repoURL string,
	branch string,
	creds *RepoCredentials,
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
	branch string,
	creds *RepoCredentials,
	updateFn func(homeDir, workingDir string) (string, error),
) (string, error) {
	repo, err := Clone(repoURL, creds)
	if err != nil {
		return "", errors.Wrapf(err, "error cloning git repo %q", repoURL)
	}
	defer repo.Close()

	if branch != "" {
		if err = repo.Checkout(branch); err != nil {
			return "", errors.Wrapf(
				err,
				"error checking out branch %q from git repo",
				repoURL,
			)
		}
	}

	commitMsg, err := updateFn(repo.HomeDir(), repo.WorkingDir())
	if err != nil {
		return "", err
	}

	var hasDiffs bool
	if hasDiffs, err = repo.HasDiffs(); err != nil || !hasDiffs {
		return "",
			errors.Wrapf(err, "error checking for diffs in git repo %q", repoURL)
	}

	if err = repo.AddAllAndCommit(commitMsg); err != nil {
		return "",
			errors.Wrapf(err, "error committing updates to git repo %q", repoURL)
	}

	if err = repo.Push(); err != nil {
		return "",
			errors.Wrapf(err, "error pushing updates to git repo %q", repoURL)
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
