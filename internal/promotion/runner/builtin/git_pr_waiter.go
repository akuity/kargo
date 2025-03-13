package builtin

import (
	"context"
	"fmt"

	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/gitprovider"
	"github.com/akuity/kargo/internal/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// gitPRWaiter is an implementation of the promotion.StepRunner interface that
// waits for a pull request to be merged or closed unmerged.
type gitPRWaiter struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
}

// newGitPRWaiter returns an implementation of the promotion.StepRunner interface
// that waits for a pull request to be merged or closed unmerged.
func newGitPRWaiter(credsDB credentials.Database) promotion.StepRunner {
	r := &gitPRWaiter{
		credsDB: credsDB,
	}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (g *gitPRWaiter) Name() string {
	return "git-wait-for-pr"
}

// Run implements the promotion.StepRunner interface.
func (g *gitPRWaiter) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	if err := g.validate(stepCtx.Config); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}
	cfg, err := promotion.ConfigToStruct[builtin.GitWaitForPRConfig](stepCtx.Config)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("could not convert config into git-wait-for-pr config: %w", err)
	}
	return g.run(ctx, stepCtx, cfg)
}

// validate validates gitPRWaiter configuration against a JSON schema.
func (g *gitPRWaiter) validate(cfg promotion.Config) error {
	return validate(g.schemaLoader, gojsonschema.NewGoLoader(cfg), g.Name())
}

func (g *gitPRWaiter) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.GitWaitForPRConfig,
) (promotion.StepResult, error) {
	var repoCreds *git.RepoCredentials
	creds, err := g.credsDB.Get(
		ctx,
		stepCtx.Project,
		credentials.TypeGit,
		cfg.RepoURL,
	)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error getting credentials for %s: %w", cfg.RepoURL, err)
	}
	if creds != nil {
		repoCreds = &git.RepoCredentials{
			Username:      creds.Username,
			Password:      creds.Password,
			SSHPrivateKey: creds.SSHPrivateKey,
		}
	}

	gpOpts := &gitprovider.Options{
		InsecureSkipTLSVerify: cfg.InsecureSkipTLSVerify,
	}
	if repoCreds != nil {
		gpOpts.Token = repoCreds.Password
	}
	if cfg.Provider != nil {
		gpOpts.Name = string(*cfg.Provider)
	}
	gitProv, err := gitprovider.New(cfg.RepoURL, gpOpts)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error creating git provider service: %w", err)
	}

	pr, err := gitProv.GetPullRequest(ctx, cfg.PRNumber)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error getting pull request %d: %w", cfg.PRNumber, err)
	}

	if pr.Open {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseRunning}, nil
	}
	if !pr.Merged {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseFailed},
			&promotion.TerminalError{Err: fmt.Errorf("pull request %d was closed without being merged", cfg.PRNumber)}
	}
	return promotion.StepResult{
		Status: kargoapi.PromotionPhaseSucceeded,
		Output: map[string]any{stateKeyCommit: pr.MergeCommitSHA},
	}, nil
}
