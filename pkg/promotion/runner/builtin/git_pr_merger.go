package builtin

import (
	"context"
	"fmt"
	"time"

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

const stepKindGitMergePR = "git-merge-pr"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindGitMergePR,
			Metadata: promotion.StepRunnerMetadata{
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityAccessCredentials,
				},
			},
			Value: newGitPRMerger,
		},
	)
}

// gitPRMerger is an implementation of the promotion.StepRunner interface that
// merges a pull request.
type gitPRMerger struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
}

// newGitPRMerger returns an implementation of the promotion.StepRunner
// interface that merges a pull request.
func newGitPRMerger(caps promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &gitPRMerger{
		credsDB:      caps.CredsDB,
		schemaLoader: getConfigSchemaLoader(stepKindGitMergePR),
	}
}

// Run implements the promotion.StepRunner interface.
func (g *gitPRMerger) Run(
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
// into a builtin.GitMergePRConfig struct.
func (g *gitPRMerger) convert(cfg promotion.Config) (builtin.GitMergePRConfig, error) {
	return validateAndConvert[builtin.GitMergePRConfig](g.schemaLoader, cfg, stepKindGitMergePR)
}

func (g *gitPRMerger) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.GitMergePRConfig,
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

	// Try to merge the PR using a primitive retry loop. PRs are often ready to
	// merge moments after being opened, but not quite immediately. Accounting
	// for this internally avoids the scenario where a Promotion needs to wait
	// for its next regularly scheduled reconciliation to merge a PR that could
	// have been merged already if we were patient for just a few seconds.
	var mergedPR *gitprovider.PullRequest
	var merged bool
	const maxMergeAttempts = 3
	for i := range maxMergeAttempts {
		if mergedPR, merged, err = gitProv.MergePullRequest(
			ctx, cfg.PRNumber,
		); err != nil {
			// Only actual errors (auth, network, invalid PR, closed but not merged,
			// etc.) reach here
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
				&promotion.TerminalError{
					Err: fmt.Errorf("error merging pull request %d: %w", cfg.PRNumber, err),
				}
		}
		if merged {
			break
		}
		if i < maxMergeAttempts {
			time.Sleep(time.Second * 5)
		}
	}

	if !merged {
		// PR is not ready to merge yet (checks pending, conflicts, etc.)
		if cfg.Wait {
			// Return RUNNING to retry later
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusRunning}, nil
		}
		// If not waiting, treat as a failure
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{
				Err: fmt.Errorf(
					"pull request %d is not ready to merge and wait is disabled",
					cfg.PRNumber,
				),
			}
	}

	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: map[string]any{stateKeyCommit: mergedPR.MergeCommitSHA},
	}, nil
}
