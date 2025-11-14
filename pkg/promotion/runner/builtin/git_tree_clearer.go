package builtin

import (
	"context"
	"fmt"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindGitClear = "git-clear"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindGitClear,
			Value: newGitTreeClearer,
		},
	)
}

// gitTreeClearer is an implementation of the promotion.StepRunner interface
// that removes the content of a Git working tree.
type gitTreeClearer struct {
	schemaLoader gojsonschema.JSONLoader
}

// newGitTreeClearer returns an implementation of the promotion.StepRunner
// interface that removes the content of a Git working tree.
func newGitTreeClearer(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &gitTreeClearer{schemaLoader: getConfigSchemaLoader(stepKindGitClear)}
}

// Run implements the promotion.StepRunner interface.
func (g *gitTreeClearer) Run(
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

// convert validates gitTreeClearer configuration against a JSON schema and
// converts it into a builtin.GitClearConfig struct.
func (g *gitTreeClearer) convert(cfg promotion.Config) (builtin.GitClearConfig, error) {
	return validateAndConvert[builtin.GitClearConfig](g.schemaLoader, cfg, stepKindGitClear)
}

func (g *gitTreeClearer) run(
	_ context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.GitClearConfig,
) (promotion.StepResult, error) {
	p, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, fmt.Errorf(
			"error joining path %s with work dir %s: %w",
			cfg.Path, stepCtx.WorkDir, err,
		)
	}
	workTree, err := git.LoadWorkTree(p, nil)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error loading working tree from %s: %w", cfg.Path, err)
	}
	// workTree.Clear() won't remove any files that aren't indexed. This is a bit
	// of a hack to ensure that we don't have any untracked files in the working
	// tree so that workTree.Clear() will remove everything.
	if err = workTree.AddAll(); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error adding all files to working tree at %s: %w", cfg.Path, err)
	}
	if err = workTree.Clear(); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error clearing working tree at %s: %w", cfg.Path, err)
	}
	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
}
