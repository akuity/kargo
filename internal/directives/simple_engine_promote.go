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
		attempt      = promoCtx.Attempts
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

		// Get the PromotionStepRunner for the step.
		reg, err := e.registry.GetPromotionStepRunnerRegistration(step.Kind)
		if err != nil {
			return PromotionResult{
				Status:      kargoapi.PromotionPhaseErrored,
				CurrentStep: i,
				State:       state,
			}, err
		}

		// Check if the step has exceeded the maximum number of attempts.
		maxAttempts := step.GetMaxAttempts(reg.Runner)
		if maxAttempts > 0 && attempt >= maxAttempts {
			return PromotionResult{
				Status:      kargoapi.PromotionPhaseErrored,
				CurrentStep: i,
				State:       state,
				Attempt:     attempt,
			}, fmt.Errorf("step %q exceeded max attempts", step.Alias)
		}

		// Count the attempt we are about to make.
		attempt++

		// Execute the step.
		result, err := e.executeStep(ctx, promoCtx, step, reg, workDir, state)

		// If the step failed, and the maximum number of attempts has not been
		// reached, we are still "Running" the step and will retry it.
		if err != nil || result.Status == kargoapi.PromotionPhaseErrored || result.Status == kargoapi.PromotionPhaseFailed {
			if maxAttempts < 0 || attempt < maxAttempts {
				var message strings.Builder
				_, _ = message.WriteString(fmt.Sprintf("step %q failed (attempt %d)", step.Alias, attempt))
				if result.Message != "" {
					_, _ = message.WriteString(": ")
					_, _ = message.WriteString(result.Message)
				}
				if err != nil {
					_, _ = message.WriteString(": ")
					_, _ = message.WriteString(err.Error())
				}

				// Update the result to indicate that the step is still running.
				result.Status = kargoapi.PromotionPhaseRunning
				result.Message = message.String()

				// Swallow the error if the step failed, as we are still
				// retrying it.
				err = nil
			}
		}

		// Update the state with the step output, regardless of the result.
		state[step.Alias] = result.Output

		// If the step was not successful, return the result to wait for
		// a next attempt or to fail the promotion.
		if result.Status != kargoapi.PromotionPhaseSucceeded {
			return PromotionResult{
				Status:      result.Status,
				Message:     result.Message,
				CurrentStep: i,
				Attempt:     attempt,
				State:       state,
			}, err
		}

		// If the step was successful, reset the attempts counter and add its
		// health check to the list.
		attempt = 0
		if healthCheck := result.HealthCheckStep; healthCheck != nil {
			healthChecks = append(healthChecks, *healthCheck)
		}
	}

	// All steps have succeeded, return the final state.
	return PromotionResult{
		Status:           kargoapi.PromotionPhaseSucceeded,
		HealthCheckSteps: healthChecks,
		CurrentStep:      int64(len(steps)) - 1,
		Attempt:          0,
		State:            state,
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
