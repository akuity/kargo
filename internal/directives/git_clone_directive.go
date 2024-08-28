package directives

import (
	"context"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

func init() {
	// Register the git-clone directive with the builtins registry.
	builtins.RegisterDirective(newGitCloneDirective())
}

// gitCloneDirective is a directive that clones one or more refs from a remote
// Git repository to one or more working directories.
type gitCloneDirective struct {
	schemaLoader gojsonschema.JSONLoader
}

// newGitCloneDirective creates a new git-clone directive.
func newGitCloneDirective() Directive {
	return &gitCloneDirective{
		schemaLoader: getConfigSchemaLoader("git-clone"),
	}
}

// Name implements the Directive interface.
func (g *gitCloneDirective) Name() string {
	return "git-clone"
}

// Run implements the Directive interface.
func (g *gitCloneDirective) Run(
	_ context.Context,
	stepCtx *StepContext,
) (Result, error) {
	// Validate the configuration against the JSON Schema
	if err := validate(
		g.schemaLoader,
		gojsonschema.NewGoLoader(stepCtx.Config),
		"git-clone",
	); err != nil {
		return ResultFailure, err
	}
	if _, err := configToStruct[GitCloneConfig](stepCtx.Config); err != nil {
		return ResultFailure,
			fmt.Errorf("could not convert config into git-clone config: %w", err)
	}
	// TODO: Add implementation here
	return ResultSuccess, nil
}
