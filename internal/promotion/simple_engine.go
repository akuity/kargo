package promotion

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/health"
	"github.com/akuity/kargo/pkg/promotion"
)

// ComposeOutputStepKind is the name of the step kind that composes the output
// of a task into the shared state.
//
// This is defined here because it's a name that needs to be mutually known to
// the engine and to the built-in StepRunner that handles this step kind.
const ComposeOutputStepKind = "compose-output"

// ReservedStepAliasRegex is a regular expression that matches step aliases that
// are reserved for internal use.
var ReservedStepAliasRegex = regexp.MustCompile(`^(step|task)-\d+$`)

// simpleEngine is a simple implementation of the Engine interface that uses
// built-in StepRunners.
type simpleEngine struct {
	registry    stepRunnerRegistry
	kargoClient client.Client
}

// NewSimpleEngine returns a simple implementation of the Engine interface that
// uses built-in StepRunners.
func NewSimpleEngine(kargoClient client.Client) Engine {
	return &simpleEngine{
		registry:    stepRunnerReg,
		kargoClient: kargoClient,
	}
}

// Promote implements the Engine interface.
func (e *simpleEngine) Promote(
	ctx context.Context,
	promoCtx Context,
	steps []Step,
) (Result, error) {
	workDir, err := e.setupWorkDir(promoCtx.WorkDir)
	if err != nil {
		return Result{Status: kargoapi.PromotionPhaseErrored}, err
	}
	if workDir != promoCtx.WorkDir {
		defer os.RemoveAll(workDir)
	}

	if promoCtx.Secrets, err = e.getProjectSecrets(ctx, promoCtx.Project); err != nil {
		return Result{Status: kargoapi.PromotionPhaseErrored}, err
	}

	result, err := e.executeSteps(ctx, promoCtx, steps, workDir)
	if err != nil {
		return result, fmt.Errorf("step execution failed: %w", err)
	}

	return result, nil
}

// executeSteps executes a list of Steps in sequence.
func (e *simpleEngine) executeSteps(
	ctx context.Context,
	promoCtx Context,
	steps []Step,
	workDir string,
) (Result, error) {
	// Initialize the state which will be passed to each step.
	// This is the state that will be updated by each step,
	// and returned as the final state after all steps have
	// run.
	state := promoCtx.State.DeepCopy()
	if state == nil {
		state = make(promotion.State)
	}

	var (
		healthChecks  []health.Criteria
		err           error
		stepExecMetas = promoCtx.StepExecutionMetadata.DeepCopy()
	)

	// Execute each step in sequence, starting from the step index specified in
	// the Context if provided.
	for i := promoCtx.StartFromStep; i < int64(len(steps)); i++ {
		select {
		case <-ctx.Done():
			return Result{
				Status:                kargoapi.PromotionPhaseErrored,
				CurrentStep:           i,
				StepExecutionMetadata: stepExecMetas,
				State:                 state,
				HealthChecks:          healthChecks,
			}, ctx.Err()
		default:
		}

		step := steps[i]

		// Prepare the step for execution by setting the alias.
		if step.Alias, err = e.stepAlias(step.Alias, i); err != nil {
			return Result{
				Status:                kargoapi.PromotionPhaseErrored,
				CurrentStep:           i,
				StepExecutionMetadata: stepExecMetas,
				State:                 state,
				HealthChecks:          healthChecks,
			}, fmt.Errorf("error getting step alias for step %d: %w", i, err)
		}

		// Get the StepRunner for the step.
		runner := e.registry.getStepRunner(step.Kind)
		if runner == nil {
			return Result{
				Status:                kargoapi.PromotionPhaseErrored,
				CurrentStep:           i,
				StepExecutionMetadata: stepExecMetas,
				State:                 state,
				HealthChecks:          healthChecks,
			}, fmt.Errorf("no promotion step runner found for kind %d", i)
		}

		// If we don't have metadata for this step yet, create it.
		if int64(len(stepExecMetas)) == i {
			stepExecMetas = append(stepExecMetas, kargoapi.StepExecutionMetadata{
				Alias: step.Alias,
			})
		}
		stepExecMeta := &stepExecMetas[i]

		// Check if the step should be skipped.
		var skip bool
		if skip, err = step.Skip(ctx, e.kargoClient, promoCtx, state); err != nil {
			return Result{
				Status:                kargoapi.PromotionPhaseErrored,
				CurrentStep:           i,
				StepExecutionMetadata: stepExecMetas,
				State:                 state,
				HealthChecks:          healthChecks,
			}, fmt.Errorf("error checking if step %d should be skipped: %w", i, err)
		} else if skip {
			stepExecMeta.Status = kargoapi.PromotionStepStatusSkipped
			continue
		}

		// Execute the step
		if stepExecMeta.StartedAt == nil {
			stepExecMeta.StartedAt = ptr.To(metav1.Now())
		}
		result, err := e.executeStep(ctx, promoCtx, step, runner, workDir, state)
		stepExecMeta.Status = result.Status
		stepExecMeta.Message = result.Message

		// Update the state with the output of the step.
		state[step.Alias] = result.Output

		// TODO(hidde): until we have a better way to handle the output of steps
		// inflated from tasks, we need to apply a special treatment to the output
		// to allow it to become available under the alias of the "task".
		aliasNamespace := getAliasNamespace(step.Alias)
		if aliasNamespace != "" && runner.Name() == ComposeOutputStepKind {
			if state[aliasNamespace] == nil {
				state[aliasNamespace] = make(map[string]any)
			}
			for k, v := range result.Output {
				state[aliasNamespace].(map[string]any)[k] = v // nolint: forcetypeassert
			}
		}

		switch result.Status {
		case kargoapi.PromotionStepStatusErrored, kargoapi.PromotionStepStatusFailed,
			kargoapi.PromotionStepStatusRunning, kargoapi.PromotionStepStatusSucceeded,
			kargoapi.PromotionStepStatusSkipped: // Step runners can self-determine they should be skipped
		default:
			// Deal with statuses that no step should have returned.
			stepExecMeta.FinishedAt = ptr.To(metav1.Now())
			return Result{
				Status:                kargoapi.PromotionPhaseErrored,
				CurrentStep:           i,
				StepExecutionMetadata: stepExecMetas,
				State:                 state,
				HealthChecks:          healthChecks,
			}, fmt.Errorf("step %d returned an invalid status", i)
		}

		// Reconcile status and err...
		if err != nil {
			if stepExecMeta.Status != kargoapi.PromotionStepStatusFailed {
				// All states other than Errored and Failed should be mutually exclusive
				// with a hard error. If we got to here, a step has violated this
				// assumption. We will prioritize the error over the status and change
				// the status to Errored.
				stepExecMeta.Status = kargoapi.PromotionStepStatusErrored
			}
			// Let the hard error take precedence over the message.
			stepExecMeta.Message = err.Error()
		} else if result.Status == kargoapi.PromotionStepStatusErrored {
			// A nil err should be mutually exclusive with an Errored status. If we
			// got to here, a step has violated this assumption. We will prioritize
			// the Errored status over the nil error and create an error.
			message := stepExecMeta.Message
			if message == "" {
				message = "no details provided"
			}
			err = fmt.Errorf("step %d errored: %s", i, message)
		}

		// At this point, we've sorted out any discrepancies between the status and
		// err.

		switch {
		case stepExecMeta.Status == kargoapi.PromotionStepStatusSucceeded:
			// Best case scenario: The step succeeded.
			stepExecMeta.FinishedAt = ptr.To(metav1.Now())
			if healthCheck := result.HealthCheck; healthCheck != nil {
				healthChecks = append(healthChecks, *healthCheck)
			}
			continue // Move on to the next step
		case promotion.IsTerminal(err):
			// This is an unrecoverable error.
			stepExecMeta.FinishedAt = ptr.To(metav1.Now())
			return Result{
				Status:                kargoapi.PromotionPhaseErrored,
				CurrentStep:           i,
				StepExecutionMetadata: stepExecMetas,
				State:                 state,
				HealthChecks:          healthChecks,
			}, fmt.Errorf("an unrecoverable error occurred: %w", err)
		case err != nil:
			// If we get to here, the error is POTENTIALLY recoverable.
			stepExecMeta.ErrorCount++
			// Check if the error threshold has been met.
			errorThreshold := step.GetErrorThreshold(runner)
			if stepExecMeta.ErrorCount >= errorThreshold {
				// The error threshold has been met.
				stepExecMeta.FinishedAt = ptr.To(metav1.Now())
				return Result{
						Status:                kargoapi.PromotionPhaseErrored,
						CurrentStep:           i,
						StepExecutionMetadata: stepExecMetas,
						State:                 state,
						HealthChecks:          healthChecks,
					}, fmt.Errorf(
						"step %d met error threshold of %d: %s", i,
						errorThreshold, stepExecMeta.Message,
					)
			}
		}

		// If we get to here, the step is either Running (waiting for some external
		// condition to be met) or it Errored/Failed but did not meet the error
		// threshold. Now we need to check if the timeout has elapsed. A nil timeout
		// or any non-positive timeout interval are treated as NO timeout, although
		// a nil timeout really shouldn't happen.
		timeout := step.GetTimeout(runner)
		if timeout != nil && *timeout > 0 && metav1.Now().Sub(stepExecMeta.StartedAt.Time) > *timeout {
			// Timeout has elapsed.
			stepExecMeta.FinishedAt = ptr.To(metav1.Now())
			return Result{
				Status:                kargoapi.PromotionPhaseErrored,
				CurrentStep:           i,
				StepExecutionMetadata: stepExecMetas,
				State:                 state,
				HealthChecks:          healthChecks,
			}, fmt.Errorf("step %d timeout of %s has elapsed", i, timeout.String())
		}

		if err != nil {
			// Treat Errored/Failed as if the step is still running so that the
			// Promotion will be requeued. The step will be retried on the next
			// reconciliation.
			stepExecMeta.Message += "; step will be retried"
			return Result{
				Status:                kargoapi.PromotionPhaseRunning,
				CurrentStep:           i,
				StepExecutionMetadata: stepExecMetas,
				State:                 state,
				HealthChecks:          healthChecks,
			}, nil
		}

		// If we get to here, the step is still Running (waiting for some external
		// condition to be met).
		stepExecMeta.ErrorCount = 0 // Reset the error count
		return Result{
			Status:                kargoapi.PromotionPhaseRunning,
			CurrentStep:           i,
			StepExecutionMetadata: stepExecMetas,
			State:                 state,
			HealthChecks:          healthChecks,
		}, nil
	}

	// All steps have succeeded, return the final state.
	return Result{
		Status:                kargoapi.PromotionPhaseSucceeded,
		CurrentStep:           int64(len(steps)) - 1,
		StepExecutionMetadata: stepExecMetas,
		State:                 state,
		HealthChecks:          healthChecks,
	}, nil
}

// executeStep executes a single Step.
func (e *simpleEngine) executeStep(
	ctx context.Context,
	promoCtx Context,
	step Step,
	runner promotion.StepRunner,
	workDir string,
	state promotion.State,
) (promotion.StepResult, error) {
	stepCtx, err := e.prepareStepContext(ctx, promoCtx, step, workDir, state)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusErrored,
		}, &promotion.TerminalError{Err: err}
	}

	result, err := runner.Run(ctx, stepCtx)
	if err != nil {
		err = fmt.Errorf("failed to run step %q: %w", step.Kind, err)
	}
	return result, err
}

// prepareStepContext prepares a StepContext corresponding to the provided Step.
func (e *simpleEngine) prepareStepContext(
	ctx context.Context,
	promoCtx Context,
	step Step,
	workDir string,
	state promotion.State,
) (*promotion.StepContext, error) {
	stateCopy := state.DeepCopy()

	stepCfg, err := step.GetConfig(ctx, e.kargoClient, promoCtx, stateCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to get step config: %w", err)
	}

	return &promotion.StepContext{
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
	}, nil
}

// stepAlias returns the alias for a step. If the alias is empty, a default
// alias is returned based on the step index.
func (e *simpleEngine) stepAlias(alias string, index int64) (string, error) {
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
func (e *simpleEngine) setupWorkDir(existingDir string) (string, error) {
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
func (e *simpleEngine) getProjectSecrets(
	ctx context.Context,
	project string,
) (map[string]map[string]string, error) {
	secretList := corev1.SecretList{}
	if err := e.kargoClient.List(
		ctx,
		&secretList,
		client.InNamespace(project),
		client.MatchingLabels{
			// Newer label
			kargoapi.CredentialTypeLabelKey: kargoapi.CredentialTypeLabelGeneric,
		},
	); err != nil {
		return nil, fmt.Errorf("error listing Secrets for Project %q: %w", project, err)
	}
	secrets := secretList.Items
	if err := e.kargoClient.List(
		ctx,
		&secretList,
		client.InNamespace(project),
		client.MatchingLabels{
			// Legacy label
			kargoapi.ProjectSecretLabelKey: kargoapi.LabelTrueValue,
		},
	); err != nil {
		return nil, fmt.Errorf("error listing Secrets for Project %q: %w", project, err)
	}
	secrets = append(secrets, secretList.Items...)
	// Sort and de-dupe
	slices.SortFunc(secrets, func(lhs, rhs corev1.Secret) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})
	secrets = slices.CompactFunc(secrets, func(lhs, rhs corev1.Secret) bool {
		return lhs.Name == rhs.Name
	})
	secretsMap := make(map[string]map[string]string, len(secrets))
	for _, secret := range secrets {
		secretsMap[secret.Name] = make(map[string]string, len(secret.Data))
		for key, value := range secret.Data {
			secretsMap[secret.Name][key] = string(value)
		}
	}
	return secretsMap, nil
}
