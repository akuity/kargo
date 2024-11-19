package directives

import (
	"context"
	"encoding/json"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// CheckHealth implements the Engine interface.
func (e *SimpleEngine) CheckHealth(
	ctx context.Context,
	healthCtx HealthCheckContext,
	steps []HealthCheckStep,
) kargoapi.Health {
	status, issues, output := e.executeHealthChecks(ctx, healthCtx, steps)
	if len(output) == 0 {
		return kargoapi.Health{
			Status: status,
			Issues: issues,
		}
	}

	b, err := json.Marshal(output)
	if err != nil {
		issues = append(issues, fmt.Sprintf("failed to marshal health output: %s", err.Error()))
	}

	return kargoapi.Health{
		Status: status,
		Issues: issues,
		Output: &apiextensionsv1.JSON{Raw: b},
	}
}

// executeHealthChecks executes a list of HealthCheckSteps in sequence.
func (e *SimpleEngine) executeHealthChecks(
	ctx context.Context,
	healthCtx HealthCheckContext,
	steps []HealthCheckStep,
) (kargoapi.HealthState, []string, []State) {
	var (
		aggregatedStatus = kargoapi.HealthStateHealthy
		aggregatedIssues []string
		aggregatedOutput = make([]State, 0, len(steps))
	)

	for _, step := range steps {
		select {
		case <-ctx.Done():
			aggregatedStatus = aggregatedStatus.Merge(kargoapi.HealthStateUnknown)
			aggregatedIssues = append(aggregatedIssues, ctx.Err().Error())
			return aggregatedStatus, aggregatedIssues, aggregatedOutput
		default:
		}

		result := e.executeHealthCheck(ctx, healthCtx, step)
		aggregatedStatus = aggregatedStatus.Merge(result.Status)
		aggregatedIssues = append(aggregatedIssues, result.Issues...)

		if result.Output != nil {
			aggregatedOutput = append(aggregatedOutput, result.Output)
		}
	}

	return aggregatedStatus, aggregatedIssues, aggregatedOutput
}

// executeHealthCheck executes a single HealthCheckStep.
func (e *SimpleEngine) executeHealthCheck(
	ctx context.Context,
	healthCtx HealthCheckContext,
	step HealthCheckStep,
) HealthCheckStepResult {
	reg, err := e.registry.GetHealthCheckStepRunnerRegistration(step.Kind)
	if err != nil {
		return HealthCheckStepResult{
			Status: kargoapi.HealthStateUnknown,
			Issues: []string{
				fmt.Sprintf("no runner registered for step kind %q: %s", step.Kind, err.Error()),
			},
		}
	}

	stepCtx := e.prepareHealthCheckStepContext(healthCtx, step, reg)
	return reg.Runner.RunHealthCheckStep(ctx, stepCtx)
}

// prepareHealthCheckStepContext prepares a HealthCheckStepContext for a HealthCheckStep.
func (e *SimpleEngine) prepareHealthCheckStepContext(
	healthCtx HealthCheckContext,
	step HealthCheckStep,
	reg HealthCheckStepRunnerRegistration,
) *HealthCheckStepContext {
	stepCtx := &HealthCheckStepContext{
		Config:  step.Config.DeepCopy(),
		Project: healthCtx.Project,
		Stage:   healthCtx.Stage,
	}

	if reg.Permissions.AllowCredentialsDB {
		stepCtx.CredentialsDB = e.credentialsDB
	}
	if reg.Permissions.AllowKargoClient {
		stepCtx.KargoClient = e.kargoClient
	}
	if reg.Permissions.AllowArgoCDClient {
		stepCtx.ArgoCDClient = e.argoCDClient
	}

	return stepCtx
}
