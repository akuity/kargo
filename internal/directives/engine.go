package directives

import (
	"context"
	"fmt"
	"os"
)

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

// Engine is a simple engine that executes a list of directives in sequence.
type Engine struct {
	registry DirectiveRegistry
}

// NewEngine returns a new Engine with the provided DirectiveRegistry.
func NewEngine(registry DirectiveRegistry) *Engine {
	return &Engine{
		registry: registry,
	}
}

// Execute runs the provided list of directives in sequence.
func (e *Engine) Execute(ctx context.Context, steps []Step) (Result, error) {
	// TODO(hidde): allow the workDir to be restored from a previous execution.
	workDir, err := os.CreateTemp("", "run-")
	if err != nil {
		return ResultFailure, fmt.Errorf("temporary working directory creation failed: %w", err)
	}
	defer os.RemoveAll(workDir.Name())

	// Initialize the shared state that will be passed to each step.
	state := make(State)

	for _, d := range steps {
		select {
		case <-ctx.Done():
			return ResultFailure, ctx.Err()
		default:
			step, err := e.registry.GetDirective(d.Directive)
			if err != nil {
				return ResultFailure, fmt.Errorf("failed to get step %q: %w", d.Directive, err)
			}

			if result, err := step.Run(ctx, &StepContext{
				WorkDir:     workDir.Name(),
				SharedState: state,
				Alias:       d.Alias,
				Config:      d.Config.DeepCopy(),
			}); err != nil {
				return result, fmt.Errorf("failed to run step %q: %w", d.Directive, err)
			}
		}
	}
	return ResultSuccess, nil
}
