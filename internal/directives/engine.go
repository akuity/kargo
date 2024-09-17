package directives

import "context"

// Engine is an interface for running a list of directives.
type Engine interface {
	// Execute runs the provided list of directives in sequence.
	Execute(ctx context.Context, steps []Step) (Status, error)
}

// Step is a single step that should be executed by the Engine.
type Step struct {
	// Directive is the name of the directive to execute for this step.
	Directive string
	// Alias is an optional alias for the step, which can be used to
	// refer to its results in subsequent steps.
	Alias string
	// Config is a map of configuration values that can be passed to the step.
	Config Config
}
