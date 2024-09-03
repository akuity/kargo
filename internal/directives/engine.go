package directives

import (
	"context"
	"fmt"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/credentials"
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
	registry      DirectiveRegistry
	credentialsDB credentials.Database
	kargoClient   client.Client
	argoCDClient  client.Client
}

// NewEngine returns a new Engine with the provided DirectiveRegistry.
func NewEngine(
	registry DirectiveRegistry,
	credentialsDB credentials.Database,
	kargoClient client.Client,
	argoCDClient client.Client,
) *Engine {
	return &Engine{
		registry:      registry,
		credentialsDB: credentialsDB,
		kargoClient:   kargoClient,
		argoCDClient:  argoCDClient,
	}
}

// Execute runs the provided list of directives in sequence.
func (e *Engine) Execute(ctx context.Context, steps []Step) (Status, error) {
	// TODO(hidde): allow the workDir to be restored from a previous execution.
	workDir, err := os.MkdirTemp("", "run-")
	if err != nil {
		return StatusFailure, fmt.Errorf("temporary working directory creation failed: %w", err)
	}
	defer os.RemoveAll(workDir)

	// Initialize the shared state that will be passed to each step.
	state := make(State)

	for _, d := range steps {
		select {
		case <-ctx.Done():
			return StatusFailure, ctx.Err()
		default:
			reg, err := e.registry.GetDirectiveRegistration(d.Directive)
			if err != nil {
				return StatusFailure, fmt.Errorf("failed to get step %q: %w", d.Directive, err)
			}

			stateCopy := state.DeepCopy()

			stepCtx := &StepContext{
				WorkDir:     workDir,
				SharedState: stateCopy,
				Alias:       d.Alias,
				Config:      d.Config.DeepCopy(),
			}
			// Selectively provide these capabilities via the StepContext.
			if reg.Permissions.AllowCredentialsDB {
				stepCtx.CredentialsDB = e.credentialsDB
			}
			if reg.Permissions.AllowKargoClient {
				stepCtx.KargoClient = e.kargoClient
			}
			if reg.Permissions.AllowArgoCDClient {
				stepCtx.ArgoCDClient = e.argoCDClient
			}

			result, err := reg.Directive.Run(ctx, stepCtx)
			if err != nil {
				return result.Status, fmt.Errorf("failed to run step %q: %w", d.Directive, err)
			}

			if d.Alias != "" {
				state[d.Alias] = result.Output
			}
		}
	}
	return StatusSuccess, nil
}
