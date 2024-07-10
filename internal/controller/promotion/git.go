package promotion

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
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

// gitMechanism is an implementation of the Mechanism interface that uses Git to
// update configuration in a repository. It is easily configured to support
// different types of configuration management tools.
type gitMechanism struct {
	name   string
	client client.Client
	cfg    GitConfig
	// Overridable behaviors:
	selectUpdatesFn  func([]kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate
	doSingleUpdateFn func(
		context.Context,
		*kargoapi.Stage,
		*kargoapi.Promotion,
		*kargoapi.GitRepoUpdate,
		[]kargoapi.FreightReference,
	) (*kargoapi.PromotionStatus, []kargoapi.FreightReference, error)
	getReadRefFn func(
		context.Context,
		client.Client,
		*kargoapi.Stage,
		*kargoapi.GitRepoUpdate,
		[]kargoapi.FreightReference,
	) (string, *kargoapi.GitCommit, error)
	getAuthorFn      func() (*git.User, error)
	getCredentialsFn func(
		ctx context.Context,
		namespace string,
		repoURL string,
	) (*git.RepoCredentials, error)
	gitCommitFn func(
		ctx context.Context,
		stage *kargoapi.Stage,
		update *kargoapi.GitRepoUpdate,
		newFreight []kargoapi.FreightReference,
		readRef string,
		writeBranch string,
		repo git.Repo,
		repoCreds git.RepoCredentials,
	) (string, error)
	applyConfigManagementFn func(
		ctx context.Context,
		stage *kargoapi.Stage,
		update *kargoapi.GitRepoUpdate,
		freight []kargoapi.FreightReference,
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
	cl client.Client,
	credentialsDB credentials.Database,
	selectUpdatesFn func([]kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate,
	applyConfigManagementFn func(
		ctx context.Context,
		stage *kargoapi.Stage,
		update *kargoapi.GitRepoUpdate,
		newFreight []kargoapi.FreightReference,
		sourceCommit string,
		homeDir string,
		workingDir string,
		repoCreds git.RepoCredentials,
	) ([]string, error),
) Mechanism {
	g := &gitMechanism{
		name:   name,
		client: cl,
		cfg:    GitConfigFromEnv(),
	}
	g.selectUpdatesFn = selectUpdatesFn
	g.doSingleUpdateFn = g.doSingleUpdate
	g.getReadRefFn = getReadRef
	g.getCredentialsFn = getRepoCredentialsFn(credentialsDB)
	g.getAuthorFn = g.getAuthor
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
	promo *kargoapi.Promotion,
	newFreight []kargoapi.FreightReference,
) (*kargoapi.PromotionStatus, []kargoapi.FreightReference, error) {
	updates := g.selectUpdatesFn(stage.Spec.PromotionMechanisms.GitRepoUpdates)

	if len(updates) == 0 {
		return &kargoapi.PromotionStatus{Phase: kargoapi.PromotionPhaseSucceeded}, newFreight, nil
	}

	var newStatus *kargoapi.PromotionStatus

	logger := logging.LoggerFromContext(ctx).WithValues("name", g.name)
	logger.Debug("executing promotion mechanism")

	for i := range updates {
		update := &updates[i]
		var err error
		var otherStatus *kargoapi.PromotionStatus
		if otherStatus, newFreight, err = g.doSingleUpdateFn(
			ctx,
			stage,
			promo,
			update,
			newFreight,
		); err != nil {
			return nil, newFreight, err
		}
		newStatus = aggregateGitPromoStatus(newStatus, *otherStatus)
	}

	logger.Debug("done executing promotion mechanism")

	return newStatus, newFreight, nil
}

// doSingleUpdate updates configuration in a single Git repository by
// making a git commit with the changes. If performing a pull request
// promotion, will create a with PR for the git commit instead of
// committing directly.
func (g *gitMechanism) doSingleUpdate(
	ctx context.Context,
	stage *kargoapi.Stage,
	promo *kargoapi.Promotion,
	update *kargoapi.GitRepoUpdate,
	newFreight []kargoapi.FreightReference,
) (*kargoapi.PromotionStatus, []kargoapi.FreightReference, error) {
	readRef, commit, err := g.getReadRefFn(
		ctx,
		g.client,
		stage,
		update,
		newFreight,
	)
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
		stage,
		update,
		newFreight,
		readRef,
		commitBranch,
		repo,
		*creds,
	)
	if err != nil {
		return nil, newFreight, err
	}

	newStatus := promo.Status.DeepCopy()
	if update.PullRequest != nil {
		gpClient, err := newGitProvider(update, creds)
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

	if commit != nil && newStatus.Phase == kargoapi.PromotionPhaseSucceeded {
		commit.HealthCheckCommit = commitID
	}

	return newStatus, newFreight, nil
}

// getReadRef finds a commitID or branch name to read from in order to apply the
// provided update. It first determine if the update wants a commit from a
// specific origin. It uses this information to find the commit required to
// apply the update. If no such commit is found, the reference returned will be
// the read branch specified in the update. If a commit is found, the reference
// returned will be its ID and the commit itself will also be returned. An
// error is possible if, whilst searching for a commit, there is any ambiguity
// over the desired origin.
func getReadRef(
	ctx context.Context,
	cli client.Client,
	stage *kargoapi.Stage,
	update *kargoapi.GitRepoUpdate,
	newFreight []kargoapi.FreightReference,
) (string, *kargoapi.GitCommit, error) {
	desiredOrigin := freight.GetDesiredOrigin(stage, update)
	commit, err := freight.FindCommit(ctx, cli, stage, desiredOrigin, newFreight, update.RepoURL)
	if err != nil {
		return "", nil,
			fmt.Errorf("error finding commit from repo %q: %w", update.RepoURL, err)
	}
	if commit != nil {
		return commit.ID, commit, nil
	}
	return update.ReadBranch, nil, nil
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
		logger := logging.LoggerFromContext(ctx).WithValues("repo", repoURL)
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
	stage *kargoapi.Stage,
	update *kargoapi.GitRepoUpdate,
	newFreight []kargoapi.FreightReference,
	readRef string,
	writeBranch string,
	repo git.Repo,
	repoCreds git.RepoCredentials,
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
	if g.applyConfigManagementFn != nil {
		if changes, err = g.applyConfigManagementFn(
			ctx,
			stage,
			update,
			newFreight,
			sourceCommitID,
			repo.HomeDir(),
			repo.WorkingDir(),
			repoCreds,
		); err != nil {
			return "", err
		}
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
