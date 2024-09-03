package directives

import (
	"context"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

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
	return &gitCommitDirective{
		schemaLoader: getConfigSchemaLoader("git-commit"),
	}
}

// Name implements the Directive interface.
func (g *gitCommitDirective) Name() string {
	return "git-commit"
}

// Run implements the Directive interface.
func (g *gitCommitDirective) Run(
	_ context.Context,
	stepCtx *StepContext,
) (Result, error) {
	failure := Result{Status: StatusFailure}
	// Validate the configuration against the JSON Schema
	if err := validate(
		g.schemaLoader,
		gojsonschema.NewGoLoader(stepCtx.Config),
		"git-commit",
	); err != nil {
		return failure, err
	}
	if _, err := configToStruct[GitCommitConfig](stepCtx.Config); err != nil {
		return failure,
			fmt.Errorf("could not convert config into git-commit config: %w", err)
	}
	// TODO: Add implementation here
	return Result{Status: StatusSuccess}, nil
}
