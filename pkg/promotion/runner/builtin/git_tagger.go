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

const stepKindGitTag = "git-tag"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindGitTag,
			Value: newGitTagger,
		},
	)
}

// gitTagger is an implementation of the promotion.StepRunner interface that
// creates a tag in a local Git repository.
type gitTagTagger struct {
	schemaLoader gojsonschema.JSONLoader
}

// newGitTagger returns an implementation of the promotion.StepRunner
// interface that creates a tag in a local Git repository.
func newGitTagger(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &gitTagTagger{schemaLoader: getConfigSchemaLoader(stepKindGitTag)}
}

// Run implements the promotion.StepRunner interface.
func (g *gitTagTagger) Run(
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
// into a builtin.GitTagConfig struct.
func (g *gitTagTagger) convert(cfg promotion.Config) (builtin.GitTagConfig, error) {
	return validateAndConvert[builtin.GitTagConfig](g.schemaLoader, cfg, stepKindGitTag)
}

func (g *gitTagTagger) run(
	_ context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.GitTagConfig,
) (promotion.StepResult, error) {
	path, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, fmt.Errorf(
			"error joining path %s with work dir %s: %w",
			cfg.Path, stepCtx.WorkDir, err,
		)
	}
	workTree, err := git.LoadWorkTree(path, nil)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error loading working tree from %s: %w", cfg.Path, err)
	}
	if err = workTree.CreateTag(cfg.Tag, cfg.Message); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error creating tag %s: %w", cfg.Tag, err)
	}
	commitID, err := workTree.LastCommitID()
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error getting last commit ID: %w", err)
	}
	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: map[string]any{stateKeyCommit: commitID},
	}, nil
}
