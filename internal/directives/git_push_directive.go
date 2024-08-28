package directives

import (
	"context"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

func init() {
	// Register the git-push directive with the builtins registry.
	builtins.RegisterDirective(newGitPushDirective())
}

// gitPushDirective is a directive that pushes commits from a local Git
// repository to a remote Git repository.
type gitPushDirective struct {
	schemaLoader gojsonschema.JSONLoader
}

// newGitPushDirective creates a new git-push directive.
func newGitPushDirective() Directive {
	return &gitPushDirective{
		schemaLoader: getConfigSchemaLoader("git-push"),
	}
}

// Name implements the Directive interface.
func (g *gitPushDirective) Name() string {
	return "git-push"
}

// Run implements the Directive interface.
func (g *gitPushDirective) Run(
	_ context.Context,
	stepCtx *StepContext,
) (Result, error) {
	// Validate the configuration against the JSON Schema
	if err := validate(
		g.schemaLoader,
		gojsonschema.NewGoLoader(stepCtx.Config),
		"git-push",
	); err != nil {
		return ResultFailure, err
	}
	if _, err := configToStruct[GitPushConfig](stepCtx.Config); err != nil {
		return ResultFailure,
			fmt.Errorf("could not convert config into git-push config: %w", err)
	}
	// TODO: Add implementation here
	return ResultSuccess, nil
}
