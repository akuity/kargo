package directives

import (
	"context"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

func init() {
	// Register the git-clone directive with the builtins registry.
	builtins.RegisterDirective(
		newGitCloneDirective(),
		&DirectivePermissions{
			AllowCredentialsDB: true,
			AllowKargoClient:   true,
		},
	)
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
	failure := Result{Status: StatusFailure}
	// Validate the configuration against the JSON Schema
	if err := validate(
		g.schemaLoader,
		gojsonschema.NewGoLoader(stepCtx.Config),
		"git-clone",
	); err != nil {
		return failure, err
	}
	if _, err := configToStruct[GitCloneConfig](stepCtx.Config); err != nil {
		return failure,
			fmt.Errorf("could not convert config into git-clone config: %w", err)
	}
	// TODO: Add implementation here
	return Result{Status: StatusSuccess}, nil
}
