package builtin

import (
	"context"
	"fmt"

	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/gitprovider"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"

	_ "github.com/akuity/kargo/pkg/gitprovider/azure"     // Azure provider registration
	_ "github.com/akuity/kargo/pkg/gitprovider/bitbucket" // Bitbucket provider registration
	_ "github.com/akuity/kargo/pkg/gitprovider/gitea"     // Gitea provider registration
	_ "github.com/akuity/kargo/pkg/gitprovider/github"    // GitHub provider registration
	_ "github.com/akuity/kargo/pkg/gitprovider/gitlab"    // GitLab provider registration
)

const stepKindGitWaitForPR = "git-wait-for-pr"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindGitWaitForPR,
			Metadata: promotion.StepRunnerMetadata{
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityAccessCredentials,
				},
			},
			Value: newGitPRWaiter,
		},
	)
}

// gitPRWaiter is an implementation of the promotion.StepRunner interface that
// waits for a pull request to be merged or closed unmerged.
type gitPRWaiter struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
}

// newGitPRWaiter returns an implementation of the promotion.StepRunner interface
// that waits for a pull request to be merged or closed unmerged.
func newGitPRWaiter(caps promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &gitPRWaiter{
		credsDB:      caps.CredsDB,
		schemaLoader: getConfigSchemaLoader(stepKindGitWaitForPR),
	}
}

// Run implements the promotion.StepRunner interface.
func (g *gitPRWaiter) Run(
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

// convert validates the configuration against a JSON schema and converts it
// into a builtin.GitWaitForPRConfig struct.
func (g *gitPRWaiter) convert(cfg promotion.Config) (builtin.GitWaitForPRConfig, error) {
	return validateAndConvert[builtin.GitWaitForPRConfig](g.schemaLoader, cfg, stepKindGitWaitForPR)
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
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error creating git provider service: %w", err)
	}

	pr, err := gitProv.GetPullRequest(ctx, cfg.PRNumber)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error getting pull request %d: %w", cfg.PRNumber, err)
	}

	// Build output with PR metadata for all states
	output := buildPRWaiterOutput(pr)

	if pr.Open {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusRunning,
			Output: output,
		}, nil
	}
	if !pr.Merged {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
			Output: output,
		}, &promotion.TerminalError{Err: fmt.Errorf("pull request %d was closed without being merged", cfg.PRNumber)}
	}
	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: output,
	}, nil
}

// buildPRWaiterOutput constructs the output map with PR metadata.
// It always includes the pr object with id, url, open, and merged fields.
// The commit field is only included when a merge commit SHA is available.
func buildPRWaiterOutput(pr *gitprovider.PullRequest) map[string]any {
	output := map[string]any{
		"pr": map[string]any{
			"id":     pr.Number,
			"url":    pr.URL,
			"open":   pr.Open,
			"merged": pr.Merged,
		},
	}
	if pr.MergeCommitSHA != "" {
		output[stateKeyCommit] = pr.MergeCommitSHA
	}
	return output
}
