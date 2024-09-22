package directives

import (
	"context"
	"fmt"

	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/gitprovider"
)

func init() {
	// Register the git-wait-for-pr directive with the builtins registry.
	builtins.RegisterDirective(
		newGitWaitForPRDirective(),
		&DirectivePermissions{AllowCredentialsDB: true},
	)
}

// gitWaitForPRDirective is a directive that waits for a pull request to be
// merged or closed unmerged.
type gitWaitForPRDirective struct {
	schemaLoader gojsonschema.JSONLoader
}

// newGitWaitForPRDirective creates a new git-wait-for-pr directive.
func newGitWaitForPRDirective() Directive {
	d := &gitWaitForPRDirective{}
	d.schemaLoader = getConfigSchemaLoader(d.Name())
	return d
}

// Name implements the Directive interface.
func (g *gitWaitForPRDirective) Name() string {
	return "git-wait-for-pr"
}

// Run implements the Directive interface.
func (g *gitWaitForPRDirective) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	if err := g.validate(stepCtx.Config); err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, err
	}
	cfg, err := configToStruct[GitWaitForPRConfig](stepCtx.Config)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("could not convert config into git-wait-for-pr config: %w", err)
	}
	return g.runPromotionStep(ctx, stepCtx, cfg)
}

// RunHealthCheckStep implements the Directive interface.
func (g *gitWaitForPRDirective) RunHealthCheckStep(
	context.Context,
	*HealthCheckStepContext,
) HealthCheckStepResult {
	return HealthCheckStepResult{Status: kargoapi.HealthStateNotApplicable}
}

// validate validates the git-wait-for-pr directive configuration against the
// JSON schema.
func (g *gitWaitForPRDirective) validate(cfg Config) error {
	return validate(g.schemaLoader, gojsonschema.NewGoLoader(cfg), g.Name())
}

func (g *gitWaitForPRDirective) runPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	cfg GitWaitForPRConfig,
) (PromotionStepResult, error) {
	prNumber, err := getPRNumber(stepCtx.SharedState, cfg)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("error getting PR number: %w", err)
	}

	var repoCreds *git.RepoCredentials
	creds, found, err := stepCtx.CredentialsDB.Get(
		ctx,
		stepCtx.Project,
		credentials.TypeGit,
		cfg.RepoURL,
	)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("error getting credentials for %s: %w", cfg.RepoURL, err)
	}
	if found {
		repoCreds = &git.RepoCredentials{
			Username:      creds.Username,
			Password:      creds.Password,
			SSHPrivateKey: creds.SSHPrivateKey,
		}
	}

	gpOpts := &gitprovider.GitProviderOptions{
		InsecureSkipTLSVerify: cfg.InsecureSkipTLSVerify,
	}
	if repoCreds != nil {
		gpOpts.Token = repoCreds.Password
	}
	if cfg.Provider != nil {
		gpOpts.Name = string(*cfg.Provider)
	}
	gitProviderSvc, err := gitprovider.NewGitProviderService(cfg.RepoURL, gpOpts)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("error creating git provider service: %w", err)
	}

	pr, err := gitProviderSvc.GetPullRequest(ctx, prNumber)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("error getting pull request %d: %w", prNumber, err)
	}
	if pr.IsOpen() {
		return PromotionStepResult{Status: PromotionStatusPending}, nil
	}

	merged, err := gitProviderSvc.IsPullRequestMerged(ctx, prNumber)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, fmt.Errorf(
			"error checking if pull request %d was merged: %w",
			prNumber, err,
		)
	}
	if !merged {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("pull request %d was closed without being merged", prNumber)
	}

	return PromotionStepResult{Status: PromotionStatusSuccess}, nil
}

func getPRNumber(sharedState State, cfg GitWaitForPRConfig) (int64, error) {
	prNumber := cfg.PRNumber
	if cfg.PRNumberFromOpen != "" {
		stepOutput, exists := sharedState.Get(cfg.PRNumberFromOpen)
		if !exists {
			return 0, fmt.Errorf(
				"no output found from step with alias %q",
				cfg.PRNumberFromOpen,
			)
		}
		stepOutputState, ok := stepOutput.(State)
		if !ok {
			return 0, fmt.Errorf(
				"output from step with alias %q is not a State",
				cfg.PRNumberFromOpen,
			)
		}
		prNumberAny, exists := stepOutputState.Get(branchKey)
		if !exists {
			return 0, fmt.Errorf(
				"no PR number found in output from step with alias %q",
				cfg.PRNumberFromOpen,
			)
		}
		if prNumber, ok = prNumberAny.(int64); !ok {
			return 0, fmt.Errorf(
				"PR number in output from step with alias %q is not an int64",
				cfg.PRNumberFromOpen,
			)
		}
	}
	return prNumber, nil
}
