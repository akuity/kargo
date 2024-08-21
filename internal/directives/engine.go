package directives

import (
	"context"
	"fmt"
	"os"
)

// Directive is a single directive that should be executed by the Engine.
type Directive struct {
	// Step is the name of the step to execute.
	Step string
	// Alias is an optional alias for the step, which can be used to
	// refer to its results in subsequent steps.
	Alias string
	// Config is a map of configuration values that can be passed to the step.
	Config Config
}

// Engine is a simple engine that executes a list of directives in sequence.
type Engine struct {
	registry StepRegistry
}

// NewEngine returns a new Engine with the provided StepRegistry.
func NewEngine(registry StepRegistry) *Engine {
	return &Engine{
		registry: registry,
	}
}

// Execute runs the provided list of directives in sequence.
func (e *Engine) Execute(ctx context.Context, directives []Directive) (Result, error) {
	// TODO(hidde): allow the workDir to be restored from a previous execution.
	workDir, err := os.CreateTemp("", "directives-")
	if err != nil {
		return ResultFailure, fmt.Errorf("temporary working directory creation failed: %w", err)
	}
	defer os.RemoveAll(workDir.Name())

	// Initialize the shared state that will be passed to each step.
	state := make(State)

	for _, d := range directives {
		select {
		case <-ctx.Done():
			return ResultFailure, ctx.Err()
		default:
			step, err := e.registry.GetStep(d.Step)
			if err != nil {
				return ResultFailure, fmt.Errorf("failed to get step %q: %w", d.Step, err)
			}

			if result, err := step.Run(ctx, &StepContext{
				WorkDir:     workDir.Name(),
				SharedState: state,
				Alias:       d.Alias,
				Config:      d.Config.DeepCopy(),
			}); err != nil {
				return result, fmt.Errorf("failed to run step %q: %w", d.Step, err)
			}
		}
	}
	return ResultSuccess, nil
}
