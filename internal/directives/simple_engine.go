package directives

import (
	"context"
	"fmt"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

// SimpleEngine is a simple engine that executes a list of directives in sequence.
type SimpleEngine struct {
	registry      DirectiveRegistry
	credentialsDB credentials.Database
	kargoClient   client.Client
	argoCDClient  client.Client
}

// NewSimpleEngine returns a new SimpleEngine with the provided DirectiveRegistry.
func NewSimpleEngine(
	registry DirectiveRegistry,
	credentialsDB credentials.Database,
	kargoClient client.Client,
	argoCDClient client.Client,
) *SimpleEngine {
	return &SimpleEngine{
		registry:      registry,
		credentialsDB: credentialsDB,
		kargoClient:   kargoClient,
		argoCDClient:  argoCDClient,
	}
}

// Promote implements the Engine interface.
func (e *SimpleEngine) Promote(
	ctx context.Context,
	promoCtx PromotionContext,
	steps []PromotionStep,
) (PromotionResult, error) {
	workDir := promoCtx.WorkDir
	if workDir == "" {
		var err error
		workDir, err = os.MkdirTemp("", "run-")
		if err != nil {
			return PromotionResult{Status: PromotionStatusFailure},
				fmt.Errorf("temporary working directory creation failed: %w", err)
		}
		defer os.RemoveAll(workDir)
	}

	// Initialize the shared state that will be passed to each step.
	state := make(State)

	for _, d := range steps {
		select {
		case <-ctx.Done():
			return PromotionResult{Status: PromotionStatusFailure}, ctx.Err()
		default:
			reg, err := e.registry.GetDirectiveRegistration(d.Kind)
			if err != nil {
				return PromotionResult{Status: PromotionStatusFailure},
					fmt.Errorf("failed to get step %q: %w", d.Kind, err)
			}

			stateCopy := state.DeepCopy()

			stepCtx := &PromotionStepContext{
				WorkDir:         workDir,
				SharedState:     stateCopy,
				Alias:           d.Alias,
				Config:          d.Config.DeepCopy(),
				Project:         promoCtx.Project,
				Stage:           promoCtx.Stage,
				FreightRequests: promoCtx.FreightRequests,
				Freight:         promoCtx.Freight,
			}
			// Selectively provide these capabilities via the PromotionStepContext.
			if reg.Permissions.AllowCredentialsDB {
				stepCtx.CredentialsDB = e.credentialsDB
			}
			if reg.Permissions.AllowKargoClient {
				stepCtx.KargoClient = e.kargoClient
			}
			if reg.Permissions.AllowArgoCDClient {
				stepCtx.ArgoCDClient = e.argoCDClient
			}

			result, err := reg.Directive.RunPromotionStep(ctx, stepCtx)
			if err != nil {
				return PromotionResult{Status: PromotionStatusFailure},
					fmt.Errorf("failed to run step %q: %w", d.Kind, err)
			}

			if d.Alias != "" {
				state[d.Alias] = result.Output
			}
		}
	}
	return PromotionResult{Status: PromotionStatusSuccess}, nil
}

// CheckHealth implements the Engine interface.
func (e *SimpleEngine) CheckHealth(
	context.Context,
	HealthCheckContext,
	[]HealthCheckStep,
) kargoapi.Health {
	// TODO: Implement health checks.
	return kargoapi.Health{Status: kargoapi.HealthStateNotApplicable}
}
