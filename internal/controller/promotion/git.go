package promotion

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
)

// gitMechanism is an implementation of the Mechanism interface that uses Git to
// update configuration in a repository. It is easily configured to support
// different types of configuration management tools.
type gitMechanism struct {
	name string
	// Overridable behaviors:
	selectUpdatesFn  func([]kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate
	doSingleUpdateFn func(
		ctx context.Context,
		namespace string,
		update kargoapi.GitRepoUpdate,
		newFreight kargoapi.SimpleFreight,
	) (kargoapi.SimpleFreight, error)
	getReadRefFn func(
		update kargoapi.GitRepoUpdate,
		commits []kargoapi.GitCommit,
	) (string, int, error)
	getCredentialsFn func(
		ctx context.Context,
		namespace string,
		repoURL string,
	) (*git.RepoCredentials, error)
	gitCommitFn func(
		update kargoapi.GitRepoUpdate,
		newFreight kargoapi.SimpleFreight,
		readRef string,
		writeBranch string,
		creds *git.RepoCredentials,
	) (string, error)
	applyConfigManagementFn func(
		update kargoapi.GitRepoUpdate,
		newFreight kargoapi.SimpleFreight,
		homeDir string,
		workingDir string,
	) ([]string, error)
}

// newGitMechanism returns an implementation of the Mechanism interface that
// uses Git to update configuration in a repository. It is easily configured to
// support different types of configuration management tools by passing in
// functions that select and carry out the relevant subset of updates.
func newGitMechanism(
	name string,
	credentialsDB credentials.Database,
	selectUpdatesFn func([]kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate,
	applyConfigManagementFn func(
		update kargoapi.GitRepoUpdate,
		newFreight kargoapi.SimpleFreight,
		homeDir string,
		workingDir string,
	) ([]string, error),
) Mechanism {
	g := &gitMechanism{
		name: name,
	}
	g.selectUpdatesFn = selectUpdatesFn
	g.doSingleUpdateFn = g.doSingleUpdate
	g.getReadRefFn = getReadRef
	g.getCredentialsFn = getRepoCredentialsFn(credentialsDB)
	g.gitCommitFn = g.gitCommit
	g.applyConfigManagementFn = applyConfigManagementFn
	return g
}

// GetName implements the Mechanism interface.
func (g *gitMechanism) GetName() string {
	return g.name
}

// Promote implements the Mechanism interface.
func (g *gitMechanism) Promote(
	ctx context.Context,
	stage *kargoapi.Stage,
	newFreight kargoapi.SimpleFreight,
) (kargoapi.SimpleFreight, error) {
	updates := g.selectUpdatesFn(stage.Spec.PromotionMechanisms.GitRepoUpdates)

	if len(updates) == 0 {
		return newFreight, nil
	}

	newFreight = *newFreight.DeepCopy()

	logger := logging.LoggerFromContext(ctx)
	logger.Debugf("executing %s", g.name)

	for _, update := range updates {
		var err error
		if newFreight, err = g.doSingleUpdateFn(
			ctx,
			stage.Namespace,
			update,
			newFreight,
		); err != nil {
			return newFreight, err
		}
	}

	logger.Debugf("done executing %s", g.name)

	return newFreight, nil
}

// doSingleUpdate updates configuration in a single Git repository.
func (g *gitMechanism) doSingleUpdate(
	ctx context.Context,
	namespace string,
	update kargoapi.GitRepoUpdate,
	newFreight kargoapi.SimpleFreight,
) (kargoapi.SimpleFreight, error) {
	readRef, commitIndex, err := g.getReadRefFn(update, newFreight.Commits)
	if err != nil {
		return newFreight, err
	}

	creds, err := g.getCredentialsFn(
		ctx,
		namespace,
		update.RepoURL,
	)
	if err != nil {
		return newFreight, err
	}

	commitID, err := g.gitCommitFn(
		update,
		newFreight,
		readRef,
		update.WriteBranch,
		creds,
	)
	if err != nil {
		return newFreight, err
	}

	if commitIndex > -1 {
		newFreight.Commits[commitIndex].HealthCheckCommit = commitID
	}

	return newFreight, nil
}

// getReadRef steps through the provided slice of commits to determine if any of
// them are from the same repository referenced by the provided update. If so,
// it returns the commit ID and index of the commit in the slice. If not, it
// returns the read branch specified in the update and an pseudo-index of -1.
// The function also returns an error if the update indicates that the write
// branch is the same as the read branch, which would create a subscription
// loop, and is therefore something we wish to avoid.
func getReadRef(
	update kargoapi.GitRepoUpdate,
	commits []kargoapi.GitCommit,
) (string, int, error) {
	for i, commit := range commits {
		if commit.RepoURL == update.RepoURL {
			if update.WriteBranch == commit.Branch {
				return "", -1, errors.Errorf(
					"invalid update specified; cannot write to branch %q of repo %q "+
						"because it will form a subscription loop",
					update.RepoURL,
					update.WriteBranch,
				)
			}
			return commit.ID, i, nil
		}
	}
	return update.ReadBranch, -1, nil
}

// getRepoCredentialsFn returns a function that closes over the provided
// credentials database and, when invoked, uses that database to obtain git
// repository credentials and, if found, convert them into a format that can be
// used by the git package. If no credentials are found for the specified
// repository, then nil is returned.
func getRepoCredentialsFn(
	credentialsDB credentials.Database,
) func(
	ctx context.Context,
	namespace string,
	repoURL string,
) (*git.RepoCredentials, error) {
	return func(
		ctx context.Context,
		namespace string,
		repoURL string,
	) (*git.RepoCredentials, error) {
		creds, ok, err := credentialsDB.Get(
			ctx,
			namespace,
			credentials.TypeGit,
			repoURL,
		)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error obtaining credentials for git repo %q",
				repoURL,
			)
		}
		logger := logging.LoggerFromContext(ctx).WithField("repo", repoURL)
		if !ok {
			logger.Debug("found no credentials for git repo")
			return nil, nil
		}
		logger.Debug("obtained credentials for git repo")
		return &git.RepoCredentials{
			Username:      creds.Username,
			Password:      creds.Password,
			SSHPrivateKey: creds.SSHPrivateKey,
		}, nil
	}
}

// gitCommit clones the specified git repository using the provided credentials
// (which may be nil), checks out the specified readRef (if non-empty), applies
// the provided update function to the cloned repository, and then commits and
// pushes any changes to the specified writeBranch. The function returns the
// commit ID of the last commit made to the repository, or an error if any of
// the above fails.
func (g *gitMechanism) gitCommit(
	update kargoapi.GitRepoUpdate,
	newFreight kargoapi.SimpleFreight,
	readRef string,
	writeBranch string,
	creds *git.RepoCredentials,
) (string, error) {
	if creds == nil {
		creds = &git.RepoCredentials{}
	}
	repo, err := git.Clone(update.RepoURL, *creds, nil)
	if err != nil {
		return "", errors.Wrapf(err, "error cloning git repo %q", update.RepoURL)
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

	var changes []string
	if g.applyConfigManagementFn != nil {
		if changes, err = g.applyConfigManagementFn(
			update,
			newFreight,
			repo.HomeDir(),
			repo.WorkingDir(),
		); err != nil {
			return "", err
		}
	}
	commitMsg := buildCommitMessage(changes)

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
				update.RepoURL,
			)
		} else if !branchExists {
			if err = repo.CreateOrphanedBranch(writeBranch); err != nil {
				return "", errors.Wrapf(
					err,
					"error creating branch %q in repo %q",
					writeBranch,
					update.RepoURL,
				)
			}
		} else {
			if err = repo.Checkout(writeBranch); err != nil {
				return "", errors.Wrapf(
					err,
					"error checking out branch %q from git repo %q",
					writeBranch,
					update.RepoURL,
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
		return "", errors.Wrapf(
			err,
			"error checking for diffs in git repo %q",
			update.RepoURL,
		)
	}

	if hasDiffs {
		if err = repo.AddAllAndCommit(commitMsg); err != nil {
			return "", errors.Wrapf(
				err,
				"error committing updates to git repo %q",
				update.RepoURL,
			)
		}
		if err = repo.Push(); err != nil {
			return "", errors.Wrapf(
				err,
				"error pushing updates to git repo %q",
				update.RepoURL,
			)
		}
	}

	commitID, err := repo.LastCommitID()
	if err != nil {
		return "", errors.Wrapf(
			err,
			"error getting last commit ID from git repo %q",
			update.RepoURL,
		)
	}

	return commitID, nil
}

// moveRepoContents transplants the entire contents of the source directory
// EXCEPT for the .git subdirectory into the destination directory.
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

// deleteRepoContents deletes the entire contents of the specified directory
// EXCEPT for the .git subdirectory.
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

// buildCommitMessage constructs a commit message from the provided change
// summary. If the change summary is empty, then a generic message is returned.
// If the change summary contains only one entry, then that entry is returned as
// the commit message. Otherwise, the change summary is formatted as a bulleted
// list and returned as the commit message.
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
