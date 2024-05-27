package promotion

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/kelseyhightower/envconfig"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/fs"
	libGit "github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/logging"
)

const tmpPrefix = "repo-scrap-"

type GitConfig struct {
	Name           string `envconfig:"GITCLIENT_NAME"`
	Email          string `envconfig:"GITCLIENT_EMAIL"`
	SigningKeyType string `envconfig:"GITCLIENT_SIGNING_KEY_TYPE"`
	SigningKeyPath string `envconfig:"GITCLIENT_SIGNING_KEY_PATH"`
}

func GitConfigFromEnv() GitConfig {
	var cfg GitConfig
	envconfig.MustProcess("", &cfg)
	return cfg
}

// gitRepositories is a map of Git repositories keyed by their URL.
type gitRepositories map[string]git.Repo

// Close closes all the Git repositories in the map and returns any errors that
// occurred while closing them.
func (g gitRepositories) Close() error {
	var err []error
	for _, repo := range g {
		if cErr := repo.Close(); cErr != nil {
			err = append(err, cErr)
		}
	}
	return errors.Join(err...)
}

// WorkingDirs returns a map of Git repository URLs to their working directories.
func (g gitRepositories) WorkingDirs() map[string]string {
	workingDirs := make(map[string]string, len(g))
	for url, repo := range g {
		workingDirs[url] = repo.WorkingDir()
	}
	return workingDirs
}

// gitMechanism is an implementation of the Mechanism interface that uses Git to
// update configuration in a repository. It is easily configured to support
// different types of configuration management tools.
type gitMechanism struct {
	name string
	cfg  GitConfig
	// Overridable behaviors:
	selectUpdatesFn       func([]kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate
	cloneFreightCommitsFn func(
		ctx context.Context,
		namespace string,
		commits []kargoapi.GitCommit,
	) (gitRepositories, error)
	doSingleUpdateFn func(
		ctx context.Context,
		promo *kargoapi.Promotion,
		update kargoapi.GitRepoUpdate,
		newFreight kargoapi.FreightReference,
		freightGitRepos gitRepositories,
	) (*kargoapi.PromotionStatus, kargoapi.FreightReference, error)
	getReadRefFn func(
		update kargoapi.GitRepoUpdate,
		commits []kargoapi.GitCommit,
	) (string, int, error)
	getAuthorFn      func() (*git.User, error)
	getCredentialsFn func(
		ctx context.Context,
		namespace string,
		repoURL string,
	) (*git.RepoCredentials, error)
	gitCommitFn func(
		ctx context.Context,
		update kargoapi.GitRepoUpdate,
		newFreight kargoapi.FreightReference,
		namespace string,
		readRef string,
		writeBranch string,
		repo git.Repo,
		repoCreds git.RepoCredentials,
		freightGitRepos gitRepositories,
	) (string, error)
	applyCopyPatchesFn func(
		workingDir string,
		freightRepos map[string]string,
		update kargoapi.GitRepoUpdate,
	) ([]string, error)
	applyConfigManagementFn func(
		ctx context.Context,
		update kargoapi.GitRepoUpdate,
		newFreight kargoapi.FreightReference,
		namespace string,
		sourceCommit string,
		homeDir string,
		workingDir string,
		repoCreds git.RepoCredentials,
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
		ctx context.Context,
		update kargoapi.GitRepoUpdate,
		newFreight kargoapi.FreightReference,
		namespace string,
		sourceCommit string,
		homeDir string,
		workingDir string,
		repoCreds git.RepoCredentials,
	) ([]string, error),
) Mechanism {
	g := &gitMechanism{
		name: name,
	}
	g.cfg = GitConfigFromEnv()
	g.selectUpdatesFn = selectUpdatesFn
	g.cloneFreightCommitsFn = g.cloneFreightCommits
	g.doSingleUpdateFn = g.doSingleUpdate
	g.getReadRefFn = getReadRef
	g.getCredentialsFn = getRepoCredentialsFn(credentialsDB)
	g.getAuthorFn = g.getAuthor
	g.gitCommitFn = g.gitCommit
	g.applyCopyPatchesFn = applyCopyPatches
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
	promo *kargoapi.Promotion,
	newFreight kargoapi.FreightReference,
) (*kargoapi.PromotionStatus, kargoapi.FreightReference, error) {
	updates := g.selectUpdatesFn(stage.Spec.PromotionMechanisms.GitRepoUpdates)

	if len(updates) == 0 {
		return &kargoapi.PromotionStatus{Phase: kargoapi.PromotionPhaseSucceeded}, newFreight, nil
	}

	var newStatus *kargoapi.PromotionStatus
	newFreight = *newFreight.DeepCopy()

	logger := logging.LoggerFromContext(ctx)
	logger.Debugf("executing %s", g.name)

	// Clone the Git repositories associated with the commits from the Freight
	// that are referenced by the Copy patches in the updates, if any.
	var freightGitRepos gitRepositories
	defer freightGitRepos.Close()
	if commits := findCommitsForCopyPatches(newFreight, updates...); len(commits) > 0 {
		var err error
		if freightGitRepos, err = g.cloneFreightCommitsFn(ctx, promo.Namespace, commits); err != nil {
			return nil, newFreight, err
		}
	}

	// Perform the updates
	for _, update := range updates {
		var err error
		var otherStatus *kargoapi.PromotionStatus
		if otherStatus, newFreight, err = g.doSingleUpdateFn(
			ctx,
			promo,
			update,
			newFreight,
			freightGitRepos,
		); err != nil {
			return nil, newFreight, err
		}
		newStatus = aggregateGitPromoStatus(newStatus, *otherStatus)
	}

	logger.Debugf("done executing %s", g.name)

	return newStatus, newFreight, nil
}

// cloneFreightCommits clones the Git repositories associated with the provided
// GitCommits and returns a map of the cloned repositories keyed by their URL.
// In case of an error, the function returns a map of the repositories that were
// successfully cloned before the error occurred. The caller is responsible for
// closing the repositories in the map.
func (g *gitMechanism) cloneFreightCommits(
	ctx context.Context,
	namespace string,
	commits []kargoapi.GitCommit,
) (gitRepositories, error) {
	gitRepos := make(gitRepositories, len(commits))
	for _, commit := range commits {
		creds, err := g.getCredentialsFn(ctx, namespace, commit.RepoURL)
		if err != nil {
			return gitRepos, err
		}

		repo, err := git.Clone(
			commit.RepoURL,
			&git.ClientOptions{
				User:        &git.User{},
				Credentials: creds,
			},
			&git.CloneOptions{
				Branch:       commit.Branch,
				SingleBranch: true,
				Filter:       git.FilterBlobless,
				// TODO: figure out how to get this
				//InsecureSkipTLSVerify: commit.InsecureSkipTLSVerify,
			},
		)
		if err != nil {
			return gitRepos, fmt.Errorf("cloning git repo %q: %w", commit.RepoURL, err)
		}

		if err = repo.Checkout(commit.ID); err != nil {
			return gitRepos, fmt.Errorf("checking out commit %q in git repo %q: %w", commit.ID, commit.RepoURL, err)
		}

		gitRepos[libGit.NormalizeURL(commit.RepoURL)] = repo
	}
	return gitRepos, nil
}

// doSingleUpdate updates configuration in a single Git repository by
// making a git commit with the changes. If performing a pull request
// promotion, will create a with PR for the git commit instead of
// committing directly.
func (g *gitMechanism) doSingleUpdate(
	ctx context.Context,
	promo *kargoapi.Promotion,
	update kargoapi.GitRepoUpdate,
	newFreight kargoapi.FreightReference,
	freightGitRepos gitRepositories,
) (*kargoapi.PromotionStatus, kargoapi.FreightReference, error) {
	readRef, commitIndex, err := g.getReadRefFn(update, newFreight.Commits)
	if err != nil {
		return nil, newFreight, err
	}

	author, err := g.getAuthorFn()
	if err != nil {
		return nil, newFreight, err
	}
	if author == nil {
		author = &git.User{}
	}
	creds, err := g.getCredentialsFn(
		ctx,
		promo.Namespace,
		update.RepoURL,
	)
	if err != nil {
		return nil, newFreight, err
	}
	if creds == nil {
		creds = &git.RepoCredentials{}
	}
	repo, err := git.Clone(
		update.RepoURL,
		&git.ClientOptions{
			User:        author,
			Credentials: creds,
		},
		&git.CloneOptions{
			InsecureSkipTLSVerify: update.InsecureSkipTLSVerify,
		},
	)
	if err != nil {
		return nil, newFreight, fmt.Errorf("error cloning git repo %q: %w", update.RepoURL, err)
	}
	defer repo.Close()

	commitBranch := update.WriteBranch
	if update.PullRequest != nil {
		// When doing a PR promotion, instead of committing to writeBranch directly,
		// we commit to a temporary, PR branch, which is a child of writeBranch.
		commitBranch = pullRequestBranchName(promo.Namespace, promo.Spec.Stage)

		if getPullRequestNumberFromMetadata(promo.Status.Metadata, update.RepoURL) == -1 {
			// PR was never created. Prepare the branch for the commit
			if err = preparePullRequestBranch(repo, commitBranch, update.WriteBranch); err != nil {
				return nil, newFreight, fmt.Errorf("error preparing PR branch %q: %w", update.RepoURL, err)
			}
		}
	}

	commitID, err := g.gitCommitFn(
		ctx,
		update,
		newFreight,
		promo.Namespace,
		readRef,
		commitBranch,
		repo,
		*creds,
		freightGitRepos,
	)
	if err != nil {
		return nil, newFreight, err
	}

	newStatus := promo.Status.DeepCopy()
	if update.PullRequest != nil {
		gpClient, err := newGitProvider(update.RepoURL, update.PullRequest, creds)
		if err != nil {
			return nil, newFreight, err
		}
		commitID, newStatus, err = reconcilePullRequest(ctx, promo.Status, repo, gpClient, commitBranch, update.WriteBranch)
		if err != nil {
			return nil, newFreight, err
		}
	} else {
		// For git commit promotions, promotion is successful as soon as the commit is pushed.
		newStatus.Phase = kargoapi.PromotionPhaseSucceeded
	}

	if commitIndex > -1 && newStatus.Phase == kargoapi.PromotionPhaseSucceeded {
		newFreight.Commits[commitIndex].HealthCheckCommit = commitID
	}

	return newStatus, newFreight, nil
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
	updateRepoURL := libGit.NormalizeURL(update.RepoURL)
	for i, commit := range commits {
		if libGit.NormalizeURL(commit.RepoURL) == updateRepoURL {
			if update.WriteBranch == commit.Branch && update.PullRequest == nil {
				return "", -1, fmt.Errorf(
					"invalid update specified; cannot write to branch %q of repo %q "+
						"because it will form a subscription loop",
					updateRepoURL,
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
			return nil, fmt.Errorf("error obtaining credentials for git repo %q: %w", repoURL, err)
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

func (g *gitMechanism) getAuthor() (*git.User, error) {
	author := git.User{
		Name:  g.cfg.Name,
		Email: g.cfg.Email,
	}

	switch strings.ToLower(g.cfg.SigningKeyType) {
	case "gpg", "":
		author.SigningKeyType = git.SigningKeyTypeGPG
	default:
		return nil, fmt.Errorf(
			"unsupported signing key type: %q",
			g.cfg.SigningKeyType,
		)
	}

	if g.cfg.SigningKeyPath != "" {
		if _, err := os.Stat(g.cfg.SigningKeyPath); err != nil {
			return nil, fmt.Errorf(
				"error locating the commit author signing key path %q: %w",
				g.cfg.SigningKeyPath,
				err,
			)
		}
		author.SigningKeyPath = g.cfg.SigningKeyPath
	}

	return &author, nil
}

// gitCommit checks out the specified readRef (if non-empty), applies
// the provided update function to the cloned repository, and then commits and
// pushes any changes to the specified writeBranch. The function returns the
// commit ID of the last commit made to the repository, or an error if any of
// the above fails.
func (g *gitMechanism) gitCommit(
	ctx context.Context,
	update kargoapi.GitRepoUpdate,
	newFreight kargoapi.FreightReference,
	namespace string,
	readRef string,
	writeBranch string,
	repo git.Repo,
	repoCreds git.RepoCredentials,
	freightRepos gitRepositories,
) (string, error) {
	var err error
	// If readRef is non-empty, check out the specified commit or branch,
	// otherwise just move using the repository's default branch as the source.
	if readRef != "" {
		if err = repo.Checkout(readRef); err != nil {
			return "", fmt.Errorf("error checking out %q from git repo: %w", readRef, err)
		}
	}

	sourceCommitID, err := repo.LastCommitID()
	if err != nil {
		return "", err // TODO: Wrap this
	}

	var changes []string
	copyChanges, err := g.applyCopyPatchesFn(repo.WorkingDir(), freightRepos.WorkingDirs(), update)
	if err != nil {
		return "", err
	}
	changes = append(changes, copyChanges...)

	if g.applyConfigManagementFn != nil {
		var applyErr error
		configChanges, applyErr := g.applyConfigManagementFn(
			ctx,
			update,
			newFreight,
			namespace,
			sourceCommitID,
			repo.HomeDir(),
			repo.WorkingDir(),
			repoCreds,
		)
		if applyErr != nil {
			return "", applyErr
		}
		changes = append(changes, configChanges...)
	}
	commitMsg := buildCommitMessage(changes)

	// Sometimes we don't write to the same branch we read from...
	if readRef != writeBranch {
		var tempDir string
		tempDir, err = os.MkdirTemp("", tmpPrefix)
		if err != nil {
			return "", fmt.Errorf("error creating temp directory for pending changes: %w", err)
		}
		defer os.RemoveAll(tempDir)

		if err = moveRepoContents(repo.WorkingDir(), tempDir); err != nil {
			return "", fmt.Errorf("error moving repository working tree to temporary location: %w", err)
		}

		if err = repo.ResetHard(); err != nil {
			return "", fmt.Errorf("error resetting repository working tree: %w", err)
		}

		var branchExists bool
		if branchExists, err = repo.RemoteBranchExists(writeBranch); err != nil {
			return "", fmt.Errorf(
				"error checking for existence of branch %q in remote repo %q: %w",
				writeBranch,
				update.RepoURL,
				err,
			)
		} else if !branchExists {
			if err = repo.CreateOrphanedBranch(writeBranch); err != nil {
				return "", fmt.Errorf(
					"error creating branch %q in repo %q: %w",
					writeBranch,
					update.RepoURL,
					err,
				)
			}
		} else {
			if err = repo.Checkout(writeBranch); err != nil {
				return "", fmt.Errorf(
					"error checking out branch %q from git repo %q: %w",
					writeBranch,
					update.RepoURL,
					err,
				)
			}
		}

		if err = deleteRepoContents(repo.WorkingDir()); err != nil {
			return "", fmt.Errorf("error clearing contents from repository working tree: %w", err)
		}

		if err = moveRepoContents(tempDir, repo.WorkingDir()); err != nil {
			return "", fmt.Errorf("error restoring repository working tree from temporary location: %w", err)
		}
	}

	hasDiffs, err := repo.HasDiffs()
	if err != nil {
		return "", fmt.Errorf("error checking for diffs in git repo %q: %w", update.RepoURL, err)
	}

	if hasDiffs {
		if err = repo.AddAllAndCommit(commitMsg); err != nil {
			return "", fmt.Errorf("error committing updates to git repo %q: %w", update.RepoURL, err)
		}
		if err = repo.Push(false); err != nil {
			return "", fmt.Errorf("error pushing updates to git repo %q: %w", update.RepoURL, err)
		}
	}

	commitID, err := repo.LastCommitID()
	if err != nil {
		return "", fmt.Errorf("error getting last commit ID from git repo %q: %w", update.RepoURL, err)
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

// applyCopyPatches applies the copy patches from the updates to the given Git
// repository. The source directory for the copy operation is determined based
// on whether the patch specifies a RepoURL. If a RepoURL is specified, the
// source directory is the working directory of the corresponding repository in
// the map of Freight repositories. If no RepoURL is specified, the source
// directory is the working directory of the given repository. The function
// returns a slice of strings describing the changes made, or an error if any
// of the copy operations fail.
func applyCopyPatches(
	workingDir string,
	freightRepos map[string]string,
	update kargoapi.GitRepoUpdate,
) ([]string, error) {
	changes := make([]string, 0, len(update.Patches))
	for _, patch := range update.Patches {
		if patch.Copy == nil {
			continue
		}

		sourceDir := workingDir
		if patch.Copy.RepoURL != "" {
			var ok bool
			if sourceDir, ok = freightRepos[libGit.NormalizeURL(patch.Copy.RepoURL)]; !ok {
				return nil, fmt.Errorf("no Freight repository found for URL %q", patch.Copy.RepoURL)
			}
		}
		if err := applyCopyPatch(sourceDir, workingDir, *patch.Copy); err != nil {
			return nil, fmt.Errorf("error performing copy operation: %w", err)
		}

		if patch.Copy.RepoURL != "" {
			changes = append(changes, fmt.Sprintf(
				"copied %s from %s to %s",
				patch.Copy.Source, patch.Copy.RepoURL, patch.Copy.Destination),
			)
		} else {
			changes = append(changes, fmt.Sprintf("copied %s to %s", patch.Copy.Source, patch.Copy.Destination))
		}
	}
	return changes, nil
}

// applyCopyPatch applies a single CopyPatchOperation to the target directory.
// If the source path is a file, it is copied to the destination path. If the
// source path is a directory, it is copied recursively to the destination path.
// The function returns an error if the operation fails.
func applyCopyPatch(sourceDir, targetDir string, patch kargoapi.CopyPatchOperation) error {
	// Ensure the source path is within the repository working directory
	srcPath := filepath.Join(sourceDir, patch.Source)
	if !fs.WithinBasePath(sourceDir, srcPath) {
		return fmt.Errorf("source path %q is not within the repository root", patch.Source)
	}

	// Ensure the destination path is within the repository working directory.
	dstPath, err := securejoin.SecureJoin(targetDir, patch.Destination)
	if err != nil {
		return fmt.Errorf("error resolving destination path %q: %w", patch.Destination, err)
	}

	srcInfo, err := os.Lstat(srcPath)
	if err != nil {
		return fmt.Errorf("error getting info for source path %q: %w", patch.Source, err)
	}

	switch {
	case srcInfo.Mode().IsRegular():
		if err = os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return fmt.Errorf("error creating destination directory: %w", err)
		}
		if err = fs.CopyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("error copying file %q to %q: %w", patch.Source, patch.Destination, err)
		}
		return nil
	case srcInfo.IsDir():
		if err = os.MkdirAll(dstPath, 0o755); err != nil {
			return fmt.Errorf("error creating destination directory: %w", err)
		}
		if err = fs.CopyDir(srcPath, dstPath); err != nil {
			return fmt.Errorf("error copying directory %q to %q: %w", patch.Source, patch.Destination, err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported file type for source path %q", patch.Source)
	}
}

// findCommitsForCopyPatches returns a slice of GitCommits that are associated
// with the provided FreightReference and have a RepoURL that matches the RepoURL
// of at least one of the Copy patches in the provided GitRepoUpdates.
func findCommitsForCopyPatches(
	freight kargoapi.FreightReference,
	updates ...kargoapi.GitRepoUpdate,
) []kargoapi.GitCommit {
	// Create a map to store the RepoURLs that have copy patches
	repoURLs := make(map[string]struct{})

	// Populate the map with RepoURLs from the updates
	for _, update := range updates {
		for _, patch := range update.Patches {
			if patch.Copy != nil && patch.Copy.RepoURL != "" {
				repoURLs[libGit.NormalizeURL(patch.Copy.RepoURL)] = struct{}{}
			}
		}
	}

	// Create a slice to store the commits to be returned
	commits := make([]kargoapi.GitCommit, 0, len(repoURLs))

	// Check if the commit's RepoURL is in the map
	for _, commit := range freight.Commits {
		if _, ok := repoURLs[libGit.NormalizeURL(commit.RepoURL)]; ok {
			commits = append(commits, commit)
		}
	}

	return commits
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
