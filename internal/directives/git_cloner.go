package directives

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
	gitUser      git.User
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

// newGitCloner returns an implementation of the PromotionStepRunner interface
// that clones one or more refs from a remote Git repository to one or more
// working directories.
func newGitCloner() PromotionStepRunner {
	r := &gitCloner{
		gitUser: gitUserFromEnv(),
	}
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
	cfg, err := ConfigToStruct[GitCloneConfig](stepCtx.Config)
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
	var repoCreds *git.RepoCredentials
	creds, err := stepCtx.CredentialsDB.Get(
		ctx,
		stepCtx.Project,
		credentials.TypeGit,
		cfg.RepoURL,
	)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error getting credentials for %s: %w", cfg.RepoURL, err)
	}
	if creds != nil {
		repoCreds = &git.RepoCredentials{
			Username:      creds.Username,
			Password:      creds.Password,
			SSHPrivateKey: creds.SSHPrivateKey,
		}
	}
	repo, err := git.CloneBare(
		cfg.RepoURL,
		&git.ClientOptions{
			User:                  &g.gitUser,
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
			if err = ensureRemoteBranch(repo, ref, checkout.Create); err != nil {
				return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
					fmt.Errorf("error ensuring existence of remote branch %s: %w", ref, err)
			}
		case checkout.Commit != "":
			ref = checkout.Commit
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
