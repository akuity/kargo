package directives

import (
	"context"
	"fmt"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
)

const commitKey = "commit"

func init() {
	// Register the git-commit directive with the builtins registry.
	builtins.RegisterDirective(newGitCommitDirective(), nil)
}

// gitCommitDirective is a directive that makes a commit to a local Git
// repository.
type gitCommitDirective struct {
	schemaLoader gojsonschema.JSONLoader
}

// newGitCommitDirective creates a new git-commit directive.
func newGitCommitDirective() Directive {
	d := &gitCommitDirective{}
	d.schemaLoader = getConfigSchemaLoader(d.Name())
	return d
}

// Name implements the Directive interface.
func (g *gitCommitDirective) Name() string {
	return "git-commit"
}

// Run implements the Directive interface.
func (g *gitCommitDirective) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	if err := g.validate(stepCtx.Config); err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, err
	}
	cfg, err := configToStruct[GitCommitConfig](stepCtx.Config)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("could not convert config into %s config: %w", g.Name(), err)
	}
	return g.runPromotionStep(ctx, stepCtx, cfg)
}

// RunHealthCheckStep implements the Directive interface.
func (g *gitCommitDirective) RunHealthCheckStep(
	context.Context,
	*HealthCheckStepContext,
) HealthCheckStepResult {
	return HealthCheckStepResult{Status: kargoapi.HealthStateNotApplicable}
}

// validate validates the git-commit directive configuration against the JSON
// schema.
func (g *gitCommitDirective) validate(cfg Config) error {
	return validate(g.schemaLoader, gojsonschema.NewGoLoader(cfg), g.Name())
}

func (g *gitCommitDirective) runPromotionStep(
	_ context.Context,
	stepCtx *PromotionStepContext,
	cfg GitCommitConfig,
) (PromotionStepResult, error) {
	path, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, fmt.Errorf(
			"error joining path %s with work dir %s: %w",
			cfg.Path, stepCtx.WorkDir, err,
		)
	}
	workTree, err := git.LoadWorkTree(path, nil)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("error loading working tree from %s: %w", cfg.Path, err)
	}
	if err = workTree.AddAll(); err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("error adding all changes to working tree: %w", err)
	}
	commitMsg, err := g.buildCommitMessage(stepCtx.SharedState, cfg)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("error building commit message: %w", err)
	}
	commitOpts := &git.CommitOptions{}
	if cfg.Author != nil {
		commitOpts.Author = &git.User{}
		if cfg.Author.Name != "" {
			commitOpts.Author.Name = cfg.Author.Name
		}
		if cfg.Author.Email != "" {
			commitOpts.Author.Email = cfg.Author.Email
		}
	}
	if err = workTree.Commit(commitMsg, commitOpts); err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("error committing to working tree: %w", err)
	}
	commitID, err := workTree.LastCommitID()
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("error getting last commit ID: %w", err)
	}
	return PromotionStepResult{
		Status: PromotionStatusSuccess,
		Output: State{commitKey: commitID},
	}, nil
}

func (g *gitCommitDirective) buildCommitMessage(
	sharedState State,
	cfg GitCommitConfig,
) (string, error) {
	var commitMsg string
	if cfg.Message != "" {
		commitMsg = cfg.Message
	} else if len(cfg.MessageFrom) > 0 {
		commitMsgParts := make([]string, len(cfg.MessageFrom))
		for i, alias := range cfg.MessageFrom {
			stepOutput, exists := sharedState.Get(alias)
			if !exists {
				return "", fmt.Errorf(
					"no output found from step with alias %q; cannot construct commit "+
						"message",
					alias,
				)
			}
			stepOutputState, ok := stepOutput.(State)
			if !ok {
				return "", fmt.Errorf(
					"output from step with alias %q is not a State; cannot construct "+
						"commit message",
					alias,
				)
			}
			commitMsgPart, exists := stepOutputState.Get("commitMessage")
			if !exists {
				return "", fmt.Errorf(
					"no commit message found in output from step with alias %q; cannot "+
						"construct commit message",
					alias,
				)
			}
			if commitMsgParts[i], ok = commitMsgPart.(string); !ok {
				return "", fmt.Errorf(
					"commit message in output from step with alias %q is not a string; "+
						"cannot construct commit message",
					alias,
				)
			}
		}
		if len(commitMsgParts) == 1 {
			commitMsg = commitMsgParts[0]
		} else {
			commitMsg = "Kargo applied multiple changes\n\nIncluding:\n"
			for _, commitMsgPart := range commitMsgParts {
				commitMsg = fmt.Sprintf("%s\n  * %s", commitMsg, commitMsgPart)
			}
		}
	}
	return commitMsg, nil
}
