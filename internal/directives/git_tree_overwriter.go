package directives

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/otiai10/copy"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/logging"
)

func init() {
	builtins.RegisterPromotionStepRunner(newGitTreeOverwriter(), nil)
}

// gitTreeOverwriter is an implementation of the PromotionStepRunner interface
// that overwrites the content of a Git working tree with the content from
// another directory.
type gitTreeOverwriter struct {
	schemaLoader gojsonschema.JSONLoader
}

// newGitTreeOverwriter returns an implementation of the PromotionStepRunner
// interface that overwrites the content of a Git working tree with the content
// from another directory.
func newGitTreeOverwriter() PromotionStepRunner {
	r := &gitTreeOverwriter{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner interface.
func (g *gitTreeOverwriter) Name() string {
	return "git-overwrite"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (g *gitTreeOverwriter) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	if err := g.validate(stepCtx.Config); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}
	cfg, err := configToStruct[GitOverwriteConfig](stepCtx.Config)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("could not convert config into %s config: %w", g.Name(), err)
	}
	return g.runPromotionStep(ctx, stepCtx, cfg)
}

// validate validates gitTreeOverwriter configuration against a JSON schema.
func (g *gitTreeOverwriter) validate(cfg Config) error {
	return validate(g.schemaLoader, gojsonschema.NewGoLoader(cfg), g.Name())
}

func (g *gitTreeOverwriter) runPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	cfg GitOverwriteConfig,
) (PromotionStepResult, error) {
	inPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.InPath)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, fmt.Errorf(
			"error joining path %s with work dir %s: %w",
			cfg.InPath, stepCtx.WorkDir, err,
		)
	}
	outPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, fmt.Errorf(
			"error joining path %s with work dir %s: %w",
			cfg.OutPath, stepCtx.WorkDir, err,
		)
	}
	workTree, err := git.LoadWorkTree(outPath, nil)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error loading working tree from %s: %w", cfg.OutPath, err)
	}
	// workTree.Clear() won't remove any files that aren't indexed. This is a bit
	// of a hack to ensure that we don't have any untracked files in the working
	// tree so that workTree.Clear() will remove everything.
	if err = workTree.AddAll(); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error adding all files to working tree at %s: %w", cfg.OutPath, err)
	}
	if err = workTree.Clear(); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error clearing working tree at %s: %w", cfg.OutPath, err)
	}
	inFI, err := os.Stat(inPath)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error getting info for path %s: %w", inPath, err)
	}
	if !inFI.IsDir() {
		outPath = filepath.Join(outPath, inFI.Name())
	}
	if err = copy.Copy(
		inPath,
		outPath,
		copy.Options{
			Skip: func(_ os.FileInfo, src, _ string) (bool, error) {
				return src == filepath.Join(inPath, ".git"), nil
			},
			OnSymlink: func(src string) copy.SymlinkAction {
				logging.LoggerFromContext(ctx).Trace("ignoring symlink", "src", src)
				return copy.Skip
			},
			OnError: func(_, _ string, err error) error {
				return sanitizePathError(err, stepCtx.WorkDir)
			},
		},
	); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("failed to copy %q to %q: %w", cfg.InPath, cfg.OutPath, err)
	}
	return PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, nil
}
