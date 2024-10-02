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
	builtins.RegisterPromotionStepRunner(newGitCommitter(), nil)
}

// gitCommitter is an implementation of the PromotionStepRunner interface that
// makes a commit to a local Git repository.
type gitCommitter struct {
	schemaLoader gojsonschema.JSONLoader
}

// newGitCommitter returns an implementation of the PromotionStepRunner
// interface that makes a commit to a local Git repository.
func newGitCommitter() PromotionStepRunner {
	r := &gitCommitter{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner interface.
func (g *gitCommitter) Name() string {
	return "git-commit"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (g *gitCommitter) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	if err := g.validate(stepCtx.Config); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}
	cfg, err := configToStruct[GitCommitConfig](stepCtx.Config)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("could not convert config into %s config: %w", g.Name(), err)
	}
	return g.runPromotionStep(ctx, stepCtx, cfg)
}

// validate validates gitCommitter configuration against a JSON schema.
func (g *gitCommitter) validate(cfg Config) error {
	return validate(g.schemaLoader, gojsonschema.NewGoLoader(cfg), g.Name())
}

func (g *gitCommitter) runPromotionStep(
	_ context.Context,
	stepCtx *PromotionStepContext,
	cfg GitCommitConfig,
) (PromotionStepResult, error) {
	path, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, fmt.Errorf(
			"error joining path %s with work dir %s: %w",
			cfg.Path, stepCtx.WorkDir, err,
		)
	}
	workTree, err := git.LoadWorkTree(path, nil)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error loading working tree from %s: %w", cfg.Path, err)
	}
	if err = workTree.AddAll(); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error adding all changes to working tree: %w", err)
	}
	commitMsg, err := g.buildCommitMessage(stepCtx.SharedState, cfg)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
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
	hasDiffs, err := workTree.HasDiffs()
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error checking for diffs in working tree: %w", err)
	}
	if hasDiffs {
		if err = workTree.Commit(commitMsg, commitOpts); err != nil {
			return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
				fmt.Errorf("error committing to working tree: %w", err)
		}
	}
	commitID, err := workTree.LastCommitID()
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error getting last commit ID: %w", err)
	}
	return PromotionStepResult{
		Status: kargoapi.PromotionPhaseSucceeded,
		Output: map[string]any{commitKey: commitID},
	}, nil
}

func (g *gitCommitter) buildCommitMessage(
	sharedState State,
	cfg GitCommitConfig,
) (string, error) {
	var commitMsg string
	if cfg.Message != "" {
		commitMsg = cfg.Message
	} else if len(cfg.MessageFromSteps) > 0 {
		commitMsgParts := make([]string, len(cfg.MessageFromSteps))
		for i, alias := range cfg.MessageFromSteps {
			stepOutput, exists := sharedState.Get(alias)
			if !exists {
				return "", fmt.Errorf(
					"no output found from step with alias %q; cannot construct commit "+
						"message",
					alias,
				)
			}
			stepOutputMap, ok := stepOutput.(map[string]any)
			if !ok {
				return "", fmt.Errorf(
					"output from step with alias %q is not a map[string]any; cannot construct "+
						"commit message",
					alias,
				)
			}
			commitMsgPart, exists := stepOutputMap["commitMessage"]
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
