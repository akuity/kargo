package directives

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

// ReservedStepAliasRegex is a regular expression that matches step aliases that
// are reserved for internal use.
var ReservedStepAliasRegex = regexp.MustCompile(`^step-\d+$`)

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
			return PromotionResult{
					Status:      kargoapi.PromotionPhaseErrored,
					CurrentStep: 0,
				},
				fmt.Errorf("temporary working directory creation failed: %w", err)
		}
		defer os.RemoveAll(workDir)
	}

	// Initialize the shared state that will be passed to each step.
	state := promoCtx.State.DeepCopy()
	if state == nil {
		state = make(State)
	}
	var healthCheckSteps []HealthCheckStep

	for i := promoCtx.StartFromStep; i < int64(len(steps)); i++ {
		step := steps[i]
		select {
		case <-ctx.Done():
			return PromotionResult{
				Status:      kargoapi.PromotionPhaseErrored,
				CurrentStep: i,
				State:       state,
			}, ctx.Err()
		default:
		}
		reg, err := e.registry.GetPromotionStepRunnerRegistration(step.Kind)
		if err != nil {
			return PromotionResult{
					Status:      kargoapi.PromotionPhaseErrored,
					CurrentStep: i,
					State:       state,
				},
				fmt.Errorf("no runner registered for step kind %q: %w", step.Kind, err)
		}

		stateCopy := state.DeepCopy()

		step.Alias = strings.TrimSpace(step.Alias)
		if step.Alias == "" {
			step.Alias = fmt.Sprintf("step-%d", i)
		} else if ReservedStepAliasRegex.MatchString(step.Alias) {
			// A webhook enforces this regex as well, but we're checking here to
			// account for the possibility of EXISTING Stages with a promotionTemplate
			// containing a step with a now-reserved alias.
			return PromotionResult{
				Status:      kargoapi.PromotionPhaseErrored,
				CurrentStep: i,
				State:       state,
			}, fmt.Errorf("step alias %q is forbidden", step.Alias)
		}

		stepCtx := &PromotionStepContext{
			UIBaseURL:       promoCtx.UIBaseURL,
			WorkDir:         workDir,
			SharedState:     stateCopy,
			Alias:           step.Alias,
			Config:          step.Config.DeepCopy(),
			Project:         promoCtx.Project,
			Stage:           promoCtx.Stage,
			Promotion:       promoCtx.Promotion,
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
		state[step.Alias] = result.Output
		if err != nil {
			return PromotionResult{
					Status:      kargoapi.PromotionPhaseErrored,
					CurrentStep: i,
					State:       state,
				},
				fmt.Errorf("failed to run step %q: %w", step.Kind, err)
		}

		if result.Status != kargoapi.PromotionPhaseSucceeded {
			return PromotionResult{
				Status:      result.Status,
				Message:     result.Message,
				CurrentStep: i,
				State:       state,
			}, nil
		}

		if result.HealthCheckStep != nil {
			healthCheckSteps = append(healthCheckSteps, *result.HealthCheckStep)
		}
	}
	return PromotionResult{
		Status:           kargoapi.PromotionPhaseSucceeded,
		HealthCheckSteps: healthCheckSteps,
		CurrentStep:      int64(len(steps)) - 1,
		State:            state,
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
