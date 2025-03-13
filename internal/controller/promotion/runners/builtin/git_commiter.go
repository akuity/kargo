package builtin

import (
	"context"
	"fmt"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/controller/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runners/builtin"
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
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}
	cfg, err := promotion.ConfigToStruct[builtin.GitCommitConfig](stepCtx.Config)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
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
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored}, fmt.Errorf(
			"error joining path %s with work dir %s: %w",
			cfg.Path, stepCtx.WorkDir, err,
		)
	}
	workTree, err := git.LoadWorkTree(path, nil)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error loading working tree from %s: %w", cfg.Path, err)
	}
	if err = workTree.AddAll(); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error adding all changes to working tree: %w", err)
	}
	hasDiffs, err := workTree.HasDiffs()
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error checking for diffs in working tree: %w", err)
	}
	if hasDiffs {
		var commitMsg string
		if commitMsg, err = g.buildCommitMessage(stepCtx.SharedState, cfg); err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
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
			return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
				fmt.Errorf("error committing to working tree: %w", err)
		}
	}
	commitID, err := workTree.LastCommitID()
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error getting last commit ID: %w", err)
	}
	return promotion.StepResult{
		Status: kargoapi.PromotionPhaseSucceeded,
		Output: map[string]any{stateKeyCommit: commitID},
	}, nil
}

func (g *gitCommitter) buildCommitMessage(
	sharedState promotion.State,
	cfg builtin.GitCommitConfig,
) (string, error) {
	var commitMsg string
	if cfg.Message != "" {
		commitMsg = cfg.Message
	} else if len(cfg.MessageFromSteps) > 0 { // nolint: staticcheck
		commitMsgParts := make([]string, 0, len(cfg.MessageFromSteps)) // nolint: staticcheck
		for _, alias := range cfg.MessageFromSteps {                   // nolint: staticcheck
			stepOutput, exists := sharedState.Get(alias)
			if !exists {
				// It is valid for a previous step that MIGHT have left some output
				// (potentially including a commit message fragment) not to have done
				// so.
				continue
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
				// It is valid for a previous step that MIGHT have left behind a
				// commit message fragment not to have done so.
				continue
			}
			commitMsgPartStr, ok := commitMsgPart.(string)
			if !ok {
				return "", fmt.Errorf(
					"commit message in output from step with alias %q is not a string; "+
						"cannot construct commit message",
					alias,
				)
			}
			commitMsgParts = append(commitMsgParts, commitMsgPartStr)
		}
		if len(commitMsgParts) == 0 {
			// TODO: krancour: This message is painfully generic, but there is little
			// else we can do and empty commit messages are not allowed. It should
			// also be very rare that we get here. It would only occur if no previous
			// step that the user indicated might contribute a commit message fragment
			// actually did so -- and in that case, it is unlikely there are any
			// differences to commit, which would preclude us from ever attempting to
			// build a commit message in the first place.
			commitMsg = "Kargo made some changes"
		} else if len(commitMsgParts) == 1 {
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
