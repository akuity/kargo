package directives

import (
	"context"
	"fmt"
	"os"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
)

func init() {
	builtins.RegisterPromotionStepRunner(
		newGitCloner(),
		&StepRunnerPermissions{
			AllowCredentialsDB: true,
			AllowKargoClient:   true,
		},
	)
}

// gitCloner is an implementation of the PromotionStepRunner interface that
// clones one or more refs from a remote Git repository to one or more working
// directories.
type gitCloner struct {
	schemaLoader gojsonschema.JSONLoader
}

// newGitCloner returns an implementation of the PromotionStepRunner interface
// that clones one or more refs from a remote Git repository to one or more
// working directories.
func newGitCloner() PromotionStepRunner {
	r := &gitCloner{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner interface.
func (g *gitCloner) Name() string {
	return "git-clone"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (g *gitCloner) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	if err := g.validate(stepCtx.Config); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}
	cfg, err := configToStruct[GitCloneConfig](stepCtx.Config)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("could not convert config into %s config: %w", g.Name(), err)
	}
	return g.runPromotionStep(ctx, stepCtx, cfg)
}

// validate validates gitCloner configuration against a JSON schema.
func (g *gitCloner) validate(cfg Config) error {
	return validate(g.schemaLoader, gojsonschema.NewGoLoader(cfg), g.Name())
}

func (g *gitCloner) runPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	cfg GitCloneConfig,
) (PromotionStepResult, error) {
	mustClone, err := mustCloneRepo(stepCtx, cfg)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, fmt.Errorf(
			"error determining if repo %s must be cloned: %w", cfg.RepoURL, err,
		)
	}
	if !mustClone {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, nil
	}

	var repoCreds *git.RepoCredentials
	creds, found, err := stepCtx.CredentialsDB.Get(
		ctx,
		stepCtx.Project,
		credentials.TypeGit,
		cfg.RepoURL,
	)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error getting credentials for %s: %w", cfg.RepoURL, err)
	}
	if found {
		repoCreds = &git.RepoCredentials{
			Username:      creds.Username,
			Password:      creds.Password,
			SSHPrivateKey: creds.SSHPrivateKey,
		}
	}
	repo, err := git.CloneBare(
		cfg.RepoURL,
		&git.ClientOptions{
			Credentials:           repoCreds,
			InsecureSkipTLSVerify: cfg.InsecureSkipTLSVerify,
		},
		&git.BareCloneOptions{
			BaseDir: stepCtx.WorkDir,
		},
	)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error cloning %s: %w", cfg.RepoURL, err)
	}
	for _, checkout := range cfg.Checkout {
		var ref string
		switch {
		case checkout.Branch != "":
			ref = checkout.Branch
			if err = ensureRemoteBranch(repo, ref); err != nil {
				return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
					fmt.Errorf("error ensuring existence of remote branch %s: %w", ref, err)
			}
		case checkout.FromFreight:
			var desiredOrigin *kargoapi.FreightOrigin
			if checkout.FromOrigin != nil {
				desiredOrigin = &kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKind(checkout.FromOrigin.Kind),
					Name: checkout.FromOrigin.Name,
				}
			}
			var commit *kargoapi.GitCommit
			if commit, err = freight.FindCommit(
				ctx,
				stepCtx.KargoClient,
				stepCtx.Project,
				stepCtx.FreightRequests,
				desiredOrigin,
				stepCtx.Freight.References(),
				cfg.RepoURL,
				true, // Treat as an error if not found
			); err != nil {
				return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
					fmt.Errorf("error finding commit from repo %s: %w", cfg.RepoURL, err)
			}
			ref = commit.ID
		case checkout.Tag != "":
			ref = checkout.Tag
		}
		path, err := securejoin.SecureJoin(stepCtx.WorkDir, checkout.Path)
		if err != nil {
			return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, fmt.Errorf(
				"error joining path %s with work dir %s: %w",
				checkout.Path, stepCtx.WorkDir, err,
			)
		}
		if _, err = repo.AddWorkTree(
			path,
			&git.AddWorkTreeOptions{Ref: ref},
		); err != nil {
			return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, fmt.Errorf(
				"error adding work tree %s to repo %s: %w",
				checkout.Path, cfg.RepoURL, err,
			)
		}
	}
	// Note: We do NOT defer repo.Close() because we want to keep the repository
	// around on the FS for subsequent promotion steps to use. The Engine will
	// handle all work dir cleanup.
	return PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, nil
}

// mustCloneRepo determines if the repository must be cloned. At present, there
// is no concept of partial success or retries for PromotionStepRunners, so if
// any one working tree's path already exists, we can assume a previous attempt
// to clone the repository was fully successful. If that were not the case, this
// PromotionStepRunner would not even be executed again.
func mustCloneRepo(stepCtx *PromotionStepContext, cfg GitCloneConfig) (bool, error) {
	if len(cfg.Checkout) == 0 {
		// This shouldn't actually happen because the schema enforces this being
		// non-empty.
		return false, nil
	}
	path, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Checkout[0].Path)
	if err != nil {
		return false, fmt.Errorf(
			"error joining path %s with work dir %s: %w",
			cfg.Checkout[0].Path, stepCtx.WorkDir, err,
		)
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("error checking if path %s exists: %w", path, err)
	}
	return false, nil
}

// ensureRemoteBranch ensures the existence of a remote branch. If the branch
// does not exist, an empty orphaned branch is created and pushed to the remote.
func ensureRemoteBranch(repo git.BareRepo, branch string) error {
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
