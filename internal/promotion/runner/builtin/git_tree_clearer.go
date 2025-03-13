package builtin

import (
	"context"
	"fmt"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// gitTreeClearer is an implementation of the promotion.StepRunner interface
// that removes the content of a Git working tree.
type gitTreeClearer struct {
	schemaLoader gojsonschema.JSONLoader
}

// newGitTreeClearer returns an implementation of the promotion.StepRunner
// interface that removes the content of a Git working tree.
func newGitTreeClearer() promotion.StepRunner {
	r := &gitTreeClearer{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (g *gitTreeClearer) Name() string {
	return "git-clear"
}

// Run implements the promotion.StepRunner interface.
func (g *gitTreeClearer) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	if err := g.validate(stepCtx.Config); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}
	cfg, err := promotion.ConfigToStruct[builtin.GitClearConfig](stepCtx.Config)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("could not convert config into %s config: %w", g.Name(), err)
	}
	return g.run(ctx, stepCtx, cfg)
}

// validate validates gitTreeClearer configuration against a JSON schema.
func (g *gitTreeClearer) validate(cfg promotion.Config) error {
	return validate(g.schemaLoader, gojsonschema.NewGoLoader(cfg), g.Name())
}

func (g *gitTreeClearer) run(
	_ context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.GitClearConfig,
) (promotion.StepResult, error) {
	p, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored}, fmt.Errorf(
			"error joining path %s with work dir %s: %w",
			cfg.Path, stepCtx.WorkDir, err,
		)
	}
	workTree, err := git.LoadWorkTree(p, nil)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error loading working tree from %s: %w", cfg.Path, err)
	}
	// workTree.Clear() won't remove any files that aren't indexed. This is a bit
	// of a hack to ensure that we don't have any untracked files in the working
	// tree so that workTree.Clear() will remove everything.
	if err = workTree.AddAll(); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error adding all files to working tree at %s: %w", cfg.Path, err)
	}
	if err = workTree.Clear(); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error clearing working tree at %s: %w", cfg.Path, err)
	}
	return promotion.StepResult{Status: kargoapi.PromotionPhaseSucceeded}, nil
}
