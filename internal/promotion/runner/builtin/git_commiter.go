package builtin

import (
	"context"
	"fmt"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// stateKeyCommit is the key used to store the commit ID in the shared State.
const stateKeyCommit = "commit"

// gitCommitter is an implementation of the promotion.StepRunner interface that
// makes a commit to a local Git repository.
type gitCommitter struct {
	schemaLoader gojsonschema.JSONLoader
}

// newGitCommitter returns an implementation of the promotion.StepRunner
// interface that makes a commit to a local Git repository.
func newGitCommitter() promotion.StepRunner {
	r := &gitCommitter{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (g *gitCommitter) Name() string {
	return "git-commit"
}

// Run implements the promotion.StepRunner interface.
func (g *gitCommitter) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	if err := g.validate(stepCtx.Config); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}
	cfg, err := promotion.ConfigToStruct[builtin.GitCommitConfig](stepCtx.Config)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("could not convert config into %s config: %w", g.Name(), err)
	}
	return g.run(ctx, stepCtx, cfg)
}

// validate validates gitCommitter configuration against a JSON schema.
func (g *gitCommitter) validate(cfg promotion.Config) error {
	return validate(g.schemaLoader, gojsonschema.NewGoLoader(cfg), g.Name())
}

func (g *gitCommitter) run(
	_ context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.GitCommitConfig,
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
	if err = workTree.AddAll(); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error adding all changes to working tree: %w", err)
	}
	hasDiffs, err := workTree.HasDiffs()
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error checking for diffs in working tree: %w", err)
	}

	// only commit if diffs have been found
	if hasDiffs {
		commitOpts := &git.CommitOptions{}
		if cfg.Author != nil {
			commitOpts.Author = &git.User{
				Name:       cfg.Author.Name,
				Email:      cfg.Author.Email,
				SigningKey: cfg.Author.SigningKey,
			}
		}
		if err = workTree.Commit(cfg.Message, commitOpts); err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("error committing to working tree: %w", err)
		}
	}

	commitID, err := workTree.LastCommitID()
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error getting last commit ID: %w", err)
	}

	status := kargoapi.PromotionStepStatusSucceeded
	// if there are no diffs found, the 'git-status' step is skipped
	// since there's nothing to commit. Hence, marks the status as skipped.
	if !hasDiffs {
		status = kargoapi.PromotionStepStatusSkipped
	}
	return promotion.StepResult{
		Status: status,
		Output: map[string]any{stateKeyCommit: commitID},
	}, nil
}
