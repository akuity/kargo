package builtin

import (
	"context"
	"fmt"
	"os"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/kelseyhightower/envconfig"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// gitCloner is an implementation of the promotion.StepRunner interface that
// clones one or more refs from a remote Git repository to one or more working
// directories.
type gitCloner struct {
	gitUser      git.User
	credsDB      credentials.Database
	schemaLoader gojsonschema.JSONLoader
}

// gitUserFromEnv populates a git.User struct from environment variables.
func gitUserFromEnv() git.User {
	cfg := struct {
		Name           string `envconfig:"GITCLIENT_NAME"`
		Email          string `envconfig:"GITCLIENT_EMAIL"`
		SigningKeyType string `envconfig:"GITCLIENT_SIGNING_KEY_TYPE"`
		SigningKeyPath string `envconfig:"GITCLIENT_SIGNING_KEY_PATH"`
	}{}
	envconfig.MustProcess("", &cfg)
	return git.User{
		Name:           cfg.Name,
		Email:          cfg.Email,
		SigningKeyType: git.SigningKeyType(cfg.SigningKeyType),
		SigningKeyPath: cfg.SigningKeyPath,
	}
}

// newGitCloner returns an implementation of the promotion.StepRunner interface
// that clones one or more refs from a remote Git repository to one or more
// working directories.
func newGitCloner(credsDB credentials.Database) promotion.StepRunner {
	r := &gitCloner{
		credsDB: credsDB,
		gitUser: gitUserFromEnv(),
	}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (g *gitCloner) Name() string {
	return "git-clone"
}

// Run implements the promotion.StepRunner interface.
func (g *gitCloner) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := g.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return g.run(ctx, stepCtx, cfg)
}

func (g *gitCloner) convert(cfg promotion.Config) (builtin.GitCloneConfig, error) {
	typedCfg, err := validateAndConvert[builtin.GitCloneConfig](g.schemaLoader, cfg, g.Name())
	if err != nil {
		return builtin.GitCloneConfig{}, err
	}

	// Ensure any specified aliases are unique across all checkouts
	seen := make(map[string]struct{})
	for i, checkout := range typedCfg.Checkout {
		if checkout.As != "" {
			if _, exists := seen[checkout.As]; exists {
				return builtin.GitCloneConfig{},
					fmt.Errorf("duplicate checkout alias %q at checkout[%d]", checkout.As, i)
			}
			seen[checkout.As] = struct{}{}
		}
	}

	return typedCfg, nil
}

func (g *gitCloner) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.GitCloneConfig,
) (promotion.StepResult, error) {
	var repoCreds *git.RepoCredentials
	creds, err := g.credsDB.Get(
		ctx,
		stepCtx.Project,
		credentials.TypeGit,
		cfg.RepoURL,
	)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error getting credentials for %s: %w", cfg.RepoURL, err)
	}
	if creds != nil {
		repoCreds = &git.RepoCredentials{
			Username:      creds.Username,
			Password:      creds.Password,
			SSHPrivateKey: creds.SSHPrivateKey,
		}
	}

	var repoUser git.User
	if cfg.Author != nil {
		repoUser = git.User{
			Name:       cfg.Author.Name,
			Email:      cfg.Author.Email,
			SigningKey: cfg.Author.SigningKey, // Optional, may be empty
		}
	} else {
		repoUser = g.gitUser // Default to the system-level gitUser
	}

	repo, err := git.CloneBare(
		cfg.RepoURL,
		&git.ClientOptions{
			User:                  &repoUser,
			Credentials:           repoCreds,
			InsecureSkipTLSVerify: cfg.InsecureSkipTLSVerify,
		},
		&git.BareCloneOptions{
			BaseDir: stepCtx.WorkDir,
		},
	)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error cloning %s: %w", cfg.RepoURL, err)
	}
	commits := make(map[string]any)
	for _, checkout := range cfg.Checkout {
		var ref string
		switch {
		case checkout.Branch != "":
			ref = checkout.Branch
			if err = ensureRemoteBranch(repo, ref, checkout.Create); err != nil {
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
					fmt.Errorf("error ensuring existence of remote branch %s: %w", ref, err)
			}
		case checkout.Commit != "":
			ref = checkout.Commit
		case checkout.Tag != "":
			ref = checkout.Tag
		}
		path, err := securejoin.SecureJoin(stepCtx.WorkDir, checkout.Path)
		if err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, fmt.Errorf(
				"error joining path %s with work dir %s: %w",
				checkout.Path, stepCtx.WorkDir, err,
			)
		}
		worktree, err := repo.AddWorkTree(
			path,
			&git.AddWorkTreeOptions{Ref: ref},
		)
		if err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf(
					"error adding work tree %s to repo %s: %w",
					checkout.Path, cfg.RepoURL, err,
				)
		}
		key := checkout.Path
		if checkout.As != "" {
			key = checkout.As
		}
		if commits[key], err = worktree.LastCommitID(); err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("error resolving HEAD for worktree at %s: %w", path, err)
		}
	}
	// Note: We do NOT defer repo.Close() because we want to keep the repository
	// around on the FS for subsequent promotion steps to use. The Engine will
	// handle all work dir cleanup.
	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: map[string]any{"commits": commits},
	}, nil
}

// ensureRemoteBranch checks for the existence of a remote branch. If the remote
// branch exists, no action is taken and nil is returned. If the branch does not
// exist and create == true, an empty orphaned branch is created and pushed to
// the remote. If the branch does not exist and create == false, an error is
// returned.
func ensureRemoteBranch(repo git.BareRepo, branch string, create bool) error {
	exists, err := repo.RemoteBranchExists(branch)
	if err != nil {
		return fmt.Errorf(
			"error checking if remote branch %q of repo %s exists: %w",
			branch, repo.URL(), err,
		)
	}
	if exists {
		return nil
	}
	if !create {
		return fmt.Errorf(
			"remote branch %q of repo %s does not exist; set create=true if you'd "+
				"like a non-existent remote branch to be automatically created at "+
				"checkout",
			branch,
			repo.URL(),
		)
	}
	tmpDir, err := os.MkdirTemp("", "repo-")
	if err != nil {
		return fmt.Errorf("error creating temporary directory: %w", err)
	}
	workTree, err := repo.AddWorkTree(tmpDir, &git.AddWorkTreeOptions{Orphan: true})
	if err != nil {
		return fmt.Errorf(
			"error adding temporary working tree for branch %q of repo %s: %w",
			branch, repo.URL(), err,
		)
	}
	defer workTree.Close()
	// `git worktree add --orphan some/path` (i.e. the preceding
	// repo.AddWorkTree() call) creates a new orphaned branch named "path". We
	// have no control over the branch name. It will always be equal to the last
	// component of the path. So, we will immediately create _another_ orphaned
	// branch with the name we really wanted before making an initial commit and
	// pushing it to the remote.
	if err = workTree.CreateOrphanedBranch(branch); err != nil {
		return err
	}
	if err = workTree.Commit(
		"Initial commit",
		&git.CommitOptions{AllowEmpty: true},
	); err != nil {
		return fmt.Errorf(
			"error making initial commit to new branch %q of repo %s: %w",
			branch, repo.URL(), err,
		)
	}
	if err = workTree.Push(&git.PushOptions{TargetBranch: branch}); err != nil {
		return fmt.Errorf(
			"error pushing initial commit to new branch %q to repo %s: %w",
			branch, repo.URL(), err,
		)
	}
	return nil
}
