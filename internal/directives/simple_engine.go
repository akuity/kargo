package directives

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

// SimpleEngine is a simple engine that executes a list of PromotionSteps in
// sequence.
type SimpleEngine struct {
	registry      *StepRunnerRegistry
	credentialsDB credentials.Database
	kargoClient   client.Client
	argoCDClient  client.Client
}

// NewSimpleEngine returns a new SimpleEngine that uses the package's built-in
// StepRunnerRegistry.
func NewSimpleEngine(
	credentialsDB credentials.Database,
	kargoClient client.Client,
	argoCDClient client.Client,
) *SimpleEngine {
	return &SimpleEngine{
		registry:      builtins,
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
	var healthCheckSteps []HealthCheckStep
	for _, step := range steps {
		select {
		case <-ctx.Done():
			return PromotionResult{Status: PromotionStatusFailure}, ctx.Err()
		default:
		}
		reg, err := e.registry.GetPromotionStepRunnerRegistration(step.Kind)
		if err != nil {
			return PromotionResult{Status: PromotionStatusFailure},
				fmt.Errorf("no runner registered for step kind %q: %w", step.Kind, err)
		}

		stateCopy := state.DeepCopy()

		stepCtx := &PromotionStepContext{
			WorkDir:         workDir,
			SharedState:     stateCopy,
			Alias:           step.Alias,
			Config:          step.Config.DeepCopy(),
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

		result, err := reg.Runner.RunPromotionStep(ctx, stepCtx)
		if err != nil {
			return PromotionResult{Status: PromotionStatusFailure},
				fmt.Errorf("failed to run step %q: %w", step.Kind, err)
		}

		if result.Status != PromotionStatusSuccess {
			return PromotionResult{Status: result.Status}, nil
		}

		if step.Alias != "" {
			state[step.Alias] = result.Output
		}

		if result.HealthCheckStep != nil {
			healthCheckSteps = append(healthCheckSteps, *result.HealthCheckStep)
		}
	}
	return PromotionResult{
		Status:           PromotionStatusSuccess,
		HealthCheckSteps: healthCheckSteps,
	}, nil
}

// CheckHealth implements the Engine interface.
func (e *SimpleEngine) CheckHealth(
	ctx context.Context,
	healthCtx HealthCheckContext,
	steps []HealthCheckStep,
) kargoapi.Health {
	healthStatus := kargoapi.HealthStateHealthy
	healthIssues := []string{}
	healthOutput := make([]State, 0, len(steps))
stepLoop:
	for _, step := range steps {
		select {
		case <-ctx.Done():
			healthStatus = healthStatus.Merge(kargoapi.HealthStateUnknown)
			healthIssues = append(healthIssues, ctx.Err().Error())
			break stepLoop
		default:
		}
		reg, err := e.registry.GetHealthCheckStepRunnerRegistration(step.Kind)
		if err != nil {
			healthStatus = healthStatus.Merge(kargoapi.HealthStateUnknown)
			healthIssues = append(
				healthIssues,
				fmt.Sprintf("no runner registered for step kind %q: %s", step.Kind, err.Error()),
			)
			continue
		}
		stepCtx := &HealthCheckStepContext{
			Config:  step.Config.DeepCopy(),
			Project: healthCtx.Project,
			Stage:   healthCtx.Stage,
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
		result := reg.Runner.RunHealthCheckStep(ctx, stepCtx)
		healthStatus = healthStatus.Merge(result.Status)
		healthIssues = append(healthIssues, result.Issues...)
		healthOutput = append(healthOutput, result.Output)
	}
	if len(healthOutput) == 0 {
		return kargoapi.Health{
			Status: healthStatus,
			Issues: healthIssues,
		}
	}
	bytes, err := json.Marshal(healthOutput)
	if err != nil {
		// Leave the status alone. Whatever it was determined to be was correct.
		healthIssues = append(
			healthIssues,
			fmt.Sprintf("failed to marshal health output: %s", err.Error()),
		)
	}
	return kargoapi.Health{
		Status: healthStatus,
		Issues: healthIssues,
		Output: &apiextensionsv1.JSON{Raw: bytes},
	}
}
