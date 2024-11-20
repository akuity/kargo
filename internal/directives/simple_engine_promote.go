package directives

import (
	"context"
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
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
		healthChecks []HealthCheckStep
		err          error
	)

	// Execute each step in sequence, starting from the step index
	// specified in the PromotionContext if provided.
	for i := promoCtx.StartFromStep; i < int64(len(steps)); i++ {
		select {
		case <-ctx.Done():
			return PromotionResult{
				Status:      kargoapi.PromotionPhaseErrored,
				CurrentStep: i,
				State:       state,
			}, ctx.Err()
		default:
		}

		// Prepare the step for execution by setting the alias.
		step := steps[i]
		if step.Alias, err = e.stepAlias(step.Alias, i); err != nil {
			return PromotionResult{
				Status:      kargoapi.PromotionPhaseErrored,
				CurrentStep: i,
				State:       state,
			}, err
		}

		// Execute the step.
		result, err := e.executeStep(ctx, promoCtx, step, workDir, state)

		// Update the state with the step output, regardless of the result.
		state[step.Alias] = result.Output

		// If the step was not successful, return the result to wait for
		// a next attempt or to fail the promotion.
		if result.Status != kargoapi.PromotionPhaseSucceeded {
			return PromotionResult{
				Status:      result.Status,
				Message:     result.Message,
				CurrentStep: i,
				State:       state,
			}, err
		}

		// If the step was successful, add its health check to the list.
		if healthCheck := result.HealthCheckStep; healthCheck != nil {
			healthChecks = append(healthChecks, *healthCheck)
		}
	}

	// All steps have succeeded, return the final state.
	return PromotionResult{
		Status:           kargoapi.PromotionPhaseSucceeded,
		HealthCheckSteps: healthChecks,
		CurrentStep:      int64(len(steps)) - 1,
		State:            state,
	}, nil
}

// executeStep executes a single PromotionStep.
func (e *SimpleEngine) executeStep(
	ctx context.Context,
	promoCtx PromotionContext,
	step PromotionStep,
	workDir string,
	state State,
) (PromotionStepResult, error) {
	reg, err := e.registry.GetPromotionStepRunnerRegistration(step.Kind)
	if err != nil {
		return PromotionStepResult{
			Status: kargoapi.PromotionPhaseErrored,
		}, err
	}

	stepCtx, err := e.preparePromotionStepContext(ctx, promoCtx, step, workDir, state, reg)
	if err != nil {
		return PromotionStepResult{
			Status: kargoapi.PromotionPhaseErrored,
		}, err
	}

	// Check if the step has exceeded the maximum number of attempts.
	attempts := step.GetAttempts(promoCtx.State)
	maxAttempts := step.GetMaxAttempts(reg.Runner)
	if maxAttempts > 0 && attempts >= maxAttempts {
		return PromotionStepResult{
			Status: kargoapi.PromotionPhaseErrored,
		}, fmt.Errorf("step %q exceeded max attempts", step.Alias)
	}

	// Run the step and record the attempt (regardless of the result).
	result, err := reg.Runner.RunPromotionStep(ctx, stepCtx)
	result.Output = step.RecordAttempt(state, result.Output)

	if err != nil {
		err = fmt.Errorf("failed to run step %q: %w", step.Kind, err)
	}

	// If the step failed, and the maximum number of attempts has not been
	// reached, we are still "Running" the step and will retry it.
	if err != nil || result.Status == kargoapi.PromotionPhaseErrored || result.Status == kargoapi.PromotionPhaseFailed {
		if maxAttempts < 0 || attempts+1 < maxAttempts {
			result.Status = kargoapi.PromotionPhaseRunning

			var message strings.Builder
			_, _ = message.WriteString(fmt.Sprintf("step %q failed (attempt %d)", step.Alias, attempts+1))
			if result.Message != "" {
				_, _ = message.WriteString(": ")
				_, _ = message.WriteString(result.Message)
			}
			if err != nil {
				_, _ = message.WriteString(": ")
				_, _ = message.WriteString(err.Error())
			}
			result.Message = message.String()

			// Swallow the error if the step is being retried.
			return result, nil
		}
	}

	return result, err
}

// preparePromotionStepContext prepares a PromotionStepContext for a PromotionStep.
func (e *SimpleEngine) preparePromotionStepContext(
	ctx context.Context,
	promoCtx PromotionContext,
	step PromotionStep,
	workDir string,
	state State,
	reg PromotionStepRunnerRegistration,
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

	if reg.Permissions.AllowCredentialsDB {
		stepCtx.CredentialsDB = e.credentialsDB
	}
	if reg.Permissions.AllowKargoClient {
		stepCtx.KargoClient = e.kargoClient
	}
	if reg.Permissions.AllowArgoCDClient {
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
