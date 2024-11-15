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
	builtins.RegisterPromotionStepRunner(
		newGitPRWaiter(),
		&StepRunnerPermissions{AllowCredentialsDB: true},
	)
}

// gitPRWaiter is an implementation of the PromotionStepRunner interface that
// waits for a pull request to be merged or closed unmerged.
type gitPRWaiter struct {
	schemaLoader gojsonschema.JSONLoader
}

// newGitPRWaiter returns an implementation of the PromotionStepRunner interface
// that waits for a pull request to be merged or closed unmerged.
func newGitPRWaiter() PromotionStepRunner {
	r := &gitPRWaiter{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner interface.
func (g *gitPRWaiter) Name() string {
	return "git-wait-for-pr"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (g *gitPRWaiter) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	if err := g.validate(stepCtx.Config); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}
	cfg, err := ConfigToStruct[GitWaitForPRConfig](stepCtx.Config)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("could not convert config into git-wait-for-pr config: %w", err)
	}
	return g.runPromotionStep(ctx, stepCtx, cfg)
}

// validate validates gitPRWaiter configuration against a JSON schema.
func (g *gitPRWaiter) validate(cfg Config) error {
	return validate(g.schemaLoader, gojsonschema.NewGoLoader(cfg), g.Name())
}

func (g *gitPRWaiter) runPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	cfg GitWaitForPRConfig,
) (PromotionStepResult, error) {
	prNumber, err := g.getPRNumber(stepCtx.SharedState, cfg)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error getting PR number from shared state: %w", err)
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
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error creating git provider service: %w", err)
	}

	pr, err := gitProv.GetPullRequest(ctx, prNumber)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error getting pull request %d: %w", prNumber, err)
	}

	if pr.Open {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseRunning}, nil
	}
	if !pr.Merged {
		return PromotionStepResult{
			Status:  kargoapi.PromotionPhaseFailed,
			Message: fmt.Sprintf("pull request %d was closed without being merged", prNumber),
		}, err
	}
	return PromotionStepResult{
		Status: kargoapi.PromotionPhaseSucceeded,
		Output: map[string]any{commitKey: pr.MergeCommitSHA},
	}, nil
}

// getPRNumber checks shared state for output from a previous step and returns
// any PR number from that output. If no such output is found, the output
// contains no PR number, or the PR number is not an int64 or float64, then an
// error is returned.
func (g *gitPRWaiter) getPRNumber(
	sharedState State,
	cfg GitWaitForPRConfig,
) (int64, error) {
	if cfg.PRNumberFromStep == "" {
		return cfg.PRNumber, nil
	}
	stepOutput, exists := sharedState.Get(cfg.PRNumberFromStep)
	if !exists {
		return 0, fmt.Errorf(
			"no output found from step with alias %q",
			cfg.PRNumberFromStep,
		)
	}
	stepOutputMap, ok := stepOutput.(map[string]any)
	if !ok {
		return 0, fmt.Errorf(
			"output from step with alias %q is not a map[string]any",
			cfg.PRNumberFromStep,
		)
	}
	prNumberAny, exists := stepOutputMap[prNumberKey]
	if !exists {
		return 0, fmt.Errorf(
			"no PR number found in output from step with alias %q",
			cfg.PRNumberFromStep,
		)
	}
	// If the state was rehydrated from PromotionStatus, which makes use of
	// apiextensions.JSON, the PR number will be a float64. Otherwise, it will be
	// an int64. We need to handle both cases.
	switch prNumber := prNumberAny.(type) {
	case int64:
		return prNumber, nil
	case float64:
		return int64(prNumber), nil
	default:
		return 0, fmt.Errorf(
			"PR number in output from step with alias %q is not an int64",
			cfg.PRNumberFromStep,
		)
	}
}
