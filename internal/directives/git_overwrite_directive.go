package directives

import (
	"context"
	"fmt"
	"os"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/otiai10/copy"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/logging"
)

func init() {
	// Register the git-overwrite directive with the builtins registry.
	builtins.RegisterDirective(newGitOverwriteDirective(), nil)
}

// gitOverwriteDirective is a directive that overwrites the content of a Git
// working tree with the content from another directory.
type gitOverwriteDirective struct {
	schemaLoader gojsonschema.JSONLoader
}

// newGitOverwriteDirective creates a new git-overwrite directive.
func newGitOverwriteDirective() Directive {
	d := &gitOverwriteDirective{}
	d.schemaLoader = getConfigSchemaLoader(d.Name())
	return d
}

// Name implements the Directive interface.
func (g *gitOverwriteDirective) Name() string {
	return "git-overwrite"
}

// Run implements the Directive interface.
func (g *gitOverwriteDirective) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	if err := g.validate(stepCtx.Config); err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, err
	}
	cfg, err := configToStruct[GitOverwriteConfig](stepCtx.Config)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("could not convert config into %s config: %w", g.Name(), err)
	}
	return g.runPromotionStep(ctx, stepCtx, cfg)
}

// RunHealthCheckStep implements the Directive interface.
func (g *gitOverwriteDirective) RunHealthCheckStep(
	context.Context,
	*HealthCheckStepContext,
) HealthCheckStepResult {
	return HealthCheckStepResult{Status: kargoapi.HealthStateNotApplicable}
}

// validate validates the git-overwrite directive configuration against the JSON
// schema.
func (g *gitOverwriteDirective) validate(cfg Config) error {
	return validate(g.schemaLoader, gojsonschema.NewGoLoader(cfg), g.Name())
}

func (g *gitOverwriteDirective) runPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	cfg GitOverwriteConfig,
) (PromotionStepResult, error) {
	inPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.InPath)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, fmt.Errorf(
			"error joining path %s with work dir %s: %w",
			cfg.InPath, stepCtx.WorkDir, err,
		)
	}
	outPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, fmt.Errorf(
			"error joining path %s with work dir %s: %w",
			cfg.OutPath, stepCtx.WorkDir, err,
		)
	}
	workTree, err := git.LoadWorkTree(outPath, nil)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("error loading working tree from %s: %w", cfg.OutPath, err)
	}
	// workTree.Clear() won't remove any files that aren't indexed. This is a bit
	// of a hack to ensure that we don't have any untracked files in the working
	// tree so that workTree.Clear() will remove everything.
	if err = workTree.AddAll(); err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("error adding all files to working tree at %s: %w", cfg.OutPath, err)
	}
	if err = workTree.Clear(); err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("error clearing working tree at %s: %w", cfg.OutPath, err)
	}
	if err = copy.Copy(
		inPath,
		outPath,
		copy.Options{
			Skip: func(srcFI os.FileInfo, _, _ string) (bool, error) {
				return srcFI.IsDir() && srcFI.Name() == ".git", nil
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
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("failed to copy %q to %q: %w", cfg.InPath, cfg.OutPath, err)
	}
	return PromotionStepResult{Status: PromotionStatusSuccess}, nil
}
