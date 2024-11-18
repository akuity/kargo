package directives

import (
	"context"
	"fmt"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
)

func init() {
	builtins.RegisterPromotionStepRunner(newGitTreeClearer(), nil)
}

// gitTreeClearer is an implementation of the PromotionStepRunner interface
// that removes the content of a Git working tree.
type gitTreeClearer struct {
	schemaLoader gojsonschema.JSONLoader
}

// newGitTreeClearer returns an implementation of the PromotionStepRunner
// interface that removes the content of a Git working tree.
func newGitTreeClearer() PromotionStepRunner {
	r := &gitTreeClearer{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner interface.
func (g *gitTreeClearer) Name() string {
	return "git-clear"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (g *gitTreeClearer) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	if err := g.validate(stepCtx.Config); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}
	cfg, err := ConfigToStruct[GitClearConfig](stepCtx.Config)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("could not convert config into %s config: %w", g.Name(), err)
	}
	return g.runPromotionStep(ctx, stepCtx, cfg)
}

// validate validates gitTreeClearer configuration against a JSON schema.
func (g *gitTreeClearer) validate(cfg Config) error {
	return validate(g.schemaLoader, gojsonschema.NewGoLoader(cfg), g.Name())
}

func (g *gitTreeClearer) runPromotionStep(
	_ context.Context,
	stepCtx *PromotionStepContext,
	cfg GitClearConfig,
) (PromotionStepResult, error) {
	p, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, fmt.Errorf(
			"error joining path %s with work dir %s: %w",
			cfg.Path, stepCtx.WorkDir, err,
		)
	}
	workTree, err := git.LoadWorkTree(p, nil)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error loading working tree from %s: %w", cfg.Path, err)
	}
	// workTree.Clear() won't remove any files that aren't indexed. This is a bit
	// of a hack to ensure that we don't have any untracked files in the working
	// tree so that workTree.Clear() will remove everything.
	if err = workTree.AddAll(); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error adding all files to working tree at %s: %w", cfg.Path, err)
	}
	if err = workTree.Clear(); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error clearing working tree at %s: %w", cfg.Path, err)
	}
	return PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, nil
}
