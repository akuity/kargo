package directives

import (
	"context"
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Promote implements the Engine interface.
func (e *SimpleEngine) Promote(
	ctx context.Context,
	promoCtx PromotionContext,
	steps []PromotionStep,
) (PromotionResult, error) {
	workDir, err := e.setupWorkDir(promoCtx.WorkDir)
	if err != nil {
		return PromotionResult{Status: kargoapi.PromotionPhaseErrored}, err
	}
	if workDir != promoCtx.WorkDir {
		defer os.RemoveAll(workDir)
	}

	if promoCtx.Secrets, err = e.getProjectSecrets(ctx, promoCtx.Project); err != nil {
		return PromotionResult{Status: kargoapi.PromotionPhaseErrored}, err
	}

	result, err := e.executeSteps(ctx, promoCtx, steps, workDir)
	if err != nil {
		return result, fmt.Errorf("step execution failed: %w", err)
	}

	return result, nil
}

// executeSteps executes a list of PromotionSteps in sequence.
func (e *SimpleEngine) executeSteps(
	ctx context.Context,
	promoCtx PromotionContext,
	steps []PromotionStep,
	workDir string,
) (PromotionResult, error) {
	// Initialize the state which will be passed to each step.
	// This is the state that will be updated by each step,
	// and returned as the final state after all steps have
	// run.
	state := promoCtx.State.DeepCopy()
	if state == nil {
		state = make(State)
	}

	var (
		healthChecks  []HealthCheckStep
		err           error
		stepExecMetas = promoCtx.StepExecutionMetadata.DeepCopy()
	)

	// Execute each step in sequence, starting from the step index
	// specified in the PromotionContext if provided.
	for i := promoCtx.StartFromStep; i < int64(len(steps)); i++ {
		select {
		case <-ctx.Done():
			return PromotionResult{
				Status:                kargoapi.PromotionPhaseErrored,
				CurrentStep:           i,
				StepExecutionMetadata: stepExecMetas,
				State:                 state,
				HealthCheckSteps:      healthChecks,
			}, ctx.Err()
		default:
		}

		// Prepare the step for execution by setting the alias.
		step := steps[i]
		if step.Alias, err = e.stepAlias(step.Alias, i); err != nil {
			return PromotionResult{
				Status:                kargoapi.PromotionPhaseErrored,
				CurrentStep:           i,
				StepExecutionMetadata: stepExecMetas,
				State:                 state,
				HealthCheckSteps:      healthChecks,
			}, fmt.Errorf("error getting step alias for step %d: %w", i, err)
		}

		// Get the PromotionStepRunner for the step.
		reg, err := e.registry.GetPromotionStepRunnerRegistration(step.Kind)
		if err != nil {
			return PromotionResult{
				Status:                kargoapi.PromotionPhaseErrored,
				CurrentStep:           i,
				StepExecutionMetadata: stepExecMetas,
				State:                 state,
				HealthCheckSteps:      healthChecks,
			}, fmt.Errorf("error getting runner for step %d: %w", i, err)
		}

		// If we don't have metadata for this step yet, create it.
		if int64(len(stepExecMetas)) == i {
			stepExecMetas = append(stepExecMetas, kargoapi.StepExecutionMetadata{
				Alias:     step.Alias,
				StartedAt: ptr.To(metav1.Now()),
			})
		}
		stepExecMeta := &stepExecMetas[i]

		// Execute the step
		result, err := e.executeStep(ctx, promoCtx, step, reg, workDir, state)
		if err != nil {
			// Let a hard error take precedence over the result status and message.
			stepExecMeta.Status = kargoapi.PromotionPhaseErrored
			stepExecMeta.Message = err.Error()
		} else {
			stepExecMeta.Status = result.Status
			stepExecMeta.Message = result.Message
		}
		state[step.Alias] = result.Output

		if stepExecMeta.Status == kargoapi.PromotionPhaseSucceeded {
			stepExecMeta.FinishedAt = ptr.To(metav1.Now())
			stepExecMeta.ErrorCount = 0
			if healthCheck := result.HealthCheckStep; healthCheck != nil {
				healthChecks = append(healthChecks, *healthCheck)
			}
			continue // Move on to the next step
		}

		// Treat errors and logical failures the same for now.
		// TODO(krancour): These may be handled differently in the future.
		if stepExecMeta.Status != kargoapi.PromotionPhaseRunning {
			stepExecMeta.ErrorCount++
			// Check if the error threshold has been met.
			errorThreshold := step.GetErrorThreshold(reg.Runner)
			if stepExecMeta.ErrorCount >= errorThreshold {
				// The error threshold has been met.
				stepExecMeta.FinishedAt = ptr.To(metav1.Now())
				return PromotionResult{
						Status:                kargoapi.PromotionPhaseErrored,
						CurrentStep:           i,
						StepExecutionMetadata: stepExecMetas,
						State:                 state,
						HealthCheckSteps:      healthChecks,
					}, fmt.Errorf(
						"step %d met error threshold of %d: %s", i,
						errorThreshold, stepExecMeta.Message,
					)
			}
		}

		// If we get to here, the step is either running (waiting for some external
		// condition to be met) or it errored/failed but did not meet the error
		// threshold. Now we need to check if the timeout has elapsed. A nil timeout
		// or any non-positive timeout interval are treated as NO timeout, although
		// a nil timeout really shouldn't happen.
		timeout := step.GetTimeout(reg.Runner)
		if timeout != nil && *timeout > 0 && metav1.Now().Sub(stepExecMeta.StartedAt.Time) > *timeout {
			// Timeout has elapsed.
			stepExecMeta.FinishedAt = ptr.To(metav1.Now())
			return PromotionResult{
				Status:                kargoapi.PromotionPhaseErrored,
				CurrentStep:           i,
				StepExecutionMetadata: stepExecMetas,
				State:                 state,
				HealthCheckSteps:      healthChecks,
			}, fmt.Errorf("step %d timeout of %s has elapsed", i, timeout.String())
		}

		if stepExecMeta.Status != kargoapi.PromotionPhaseRunning {
			// Treat the error/failure as if the step is still running so that the
			// Promotion will be requeued. The step will be retried on the next
			// reconciliation.
			stepExecMeta.Message += "; step will be retried"
			return PromotionResult{
				Status:                kargoapi.PromotionPhaseRunning,
				CurrentStep:           i,
				StepExecutionMetadata: stepExecMetas,
				State:                 state,
				HealthCheckSteps:      healthChecks,
			}, nil
		}

		// If we get to here, the step is still running (waiting for some external
		// condition to be met).
		stepExecMeta.ErrorCount = 0 // Reset the error count
		return PromotionResult{
			Status:                kargoapi.PromotionPhaseRunning,
			CurrentStep:           i,
			StepExecutionMetadata: stepExecMetas,
			State:                 state,
			HealthCheckSteps:      healthChecks,
		}, nil
	}

	// All steps have succeeded, return the final state.
	return PromotionResult{
		Status:                kargoapi.PromotionPhaseSucceeded,
		CurrentStep:           int64(len(steps)) - 1,
		StepExecutionMetadata: stepExecMetas,
		State:                 state,
		HealthCheckSteps:      healthChecks,
	}, nil
}

// executeStep executes a single PromotionStep.
func (e *SimpleEngine) executeStep(
	ctx context.Context,
	promoCtx PromotionContext,
	step PromotionStep,
	reg PromotionStepRunnerRegistration,
	workDir string,
	state State,
) (PromotionStepResult, error) {
	stepCtx, err := e.preparePromotionStepContext(ctx, promoCtx, step, reg.Permissions, workDir, state)
	if err != nil {
		// TODO(krancour): We're not yet distinguishing between retryable and
		// non-retryable errors. When we start to do this, failure to prepare the
		// step context (likely due to invalid configuration) should be considered
		// non-retryable.
		return PromotionStepResult{
			Status: kargoapi.PromotionPhaseErrored,
		}, err
	}

	result, err := reg.Runner.RunPromotionStep(ctx, stepCtx)
	if err != nil {
		err = fmt.Errorf("failed to run step %q: %w", step.Kind, err)
	}
	return result, err
}

// preparePromotionStepContext prepares a PromotionStepContext for a PromotionStep.
func (e *SimpleEngine) preparePromotionStepContext(
	ctx context.Context,
	promoCtx PromotionContext,
	step PromotionStep,
	permissions StepRunnerPermissions,
	workDir string,
	state State,
) (*PromotionStepContext, error) {
	stateCopy := state.DeepCopy()

	stepCfg, err := step.GetConfig(ctx, e.kargoClient, promoCtx, stateCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to get step config: %w", err)
	}

	stepCtx := &PromotionStepContext{
		UIBaseURL:       promoCtx.UIBaseURL,
		WorkDir:         workDir,
		SharedState:     stateCopy,
		Alias:           step.Alias,
		Config:          stepCfg,
		Project:         promoCtx.Project,
		Stage:           promoCtx.Stage,
		Promotion:       promoCtx.Promotion,
		FreightRequests: promoCtx.FreightRequests,
		Freight:         promoCtx.Freight,
	}

	if permissions.AllowCredentialsDB {
		stepCtx.CredentialsDB = e.credentialsDB
	}
	if permissions.AllowKargoClient {
		stepCtx.KargoClient = e.kargoClient
	}
	if permissions.AllowArgoCDClient {
		stepCtx.ArgoCDClient = e.argoCDClient
	}

	return stepCtx, nil
}

// stepAlias returns the alias for a step. If the alias is empty, a default
// alias is returned based on the step index.
func (e *SimpleEngine) stepAlias(alias string, index int64) (string, error) {
	if alias = strings.TrimSpace(alias); alias != "" {
		// A webhook enforces this regex as well, but we're checking here to
		// account for the possibility of EXISTING Stages with a promotionTemplate
		// containing a step with a now-reserved alias.
		if ReservedStepAliasRegex.MatchString(alias) {
			return "", fmt.Errorf("step alias %q is forbidden", alias)
		}
		return alias, nil
	}
	return fmt.Sprintf("step-%d", index), nil
}

// setupWorkDir creates a temporary working directory if one is not provided.
func (e *SimpleEngine) setupWorkDir(existingDir string) (string, error) {
	if existingDir != "" {
		return existingDir, nil
	}

	workDir, err := os.MkdirTemp("", "run-")
	if err != nil {
		return "", fmt.Errorf("temporary working directory creation failed: %w", err)
	}
	return workDir, nil
}

// getProjectSecrets returns a map of all Secrets in the Project. The returned
// map is keyed by Secret name and contains a map of Secret data.
func (e *SimpleEngine) getProjectSecrets(
	ctx context.Context,
	project string,
) (map[string]map[string]string, error) {
	secrets := corev1.SecretList{}
	if err := e.kargoClient.List(
		ctx,
		&secrets,
		client.InNamespace(project),
	); err != nil {
		return nil, fmt.Errorf("error listing Secrets for Project %q: %w", project, err)
	}
	secretsMap := make(map[string]map[string]string, len(secrets.Items))
	for _, secret := range secrets.Items {
		secretsMap[secret.Name] = make(map[string]string, len(secret.Data))
		for key, value := range secret.Data {
			secretsMap[secret.Name][key] = string(value)
		}
	}
	return secretsMap, nil
}
