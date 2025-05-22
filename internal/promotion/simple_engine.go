package promotion

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"

	gocache "github.com/patrickmn/go-cache"
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

// ExprDataCacheFn is a function that returns a new cache to use in expression
// functions that consult the Kubernetes API.
//
// A new cache is created for each step execution, so that the cache is
// shared between all expression functions that are executed in the same step.
// This is important for performance, as our Kubernetes API client does not
// cache Secrets and ConfigMaps, but also for correctness, as the data may
// change between calls.
//
// It is allowed for the cache to be nil, in which case the expression functions
// will not cache their results.
type ExprDataCacheFn func() *gocache.Cache

// DefaultExprDataCacheFn returns a new gocache.Cache instance with
// default expiration and cleanup intervals. This is used as the default
// ExprDataCacheFn for the Engine.
func DefaultExprDataCacheFn() *gocache.Cache {
	return gocache.New(gocache.NoExpiration, gocache.NoExpiration)
}

// simpleEngine is a simple implementation of the Engine interface that uses
// built-in StepRunners.
type simpleEngine struct {
	registry    stepRunnerRegistry
	kargoClient client.Client
	cacheFunc   ExprDataCacheFn
}

// NewSimpleEngine returns a simple implementation of the Engine interface that
// uses built-in StepRunners.
func NewSimpleEngine(kargoClient client.Client, cacheFunc ExprDataCacheFn) Engine {
	return &simpleEngine{
		registry:    stepRunnerReg,
		kargoClient: kargoClient,
		cacheFunc:   cacheFunc,
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

	result := e.executeSteps(ctx, promoCtx, steps, workDir)
	if result.Status == kargoapi.PromotionPhaseErrored {
		err = errors.New(result.Message)
	}

	return result, err
}

// executeSteps executes a list of Steps in sequence.
func (e *simpleEngine) executeSteps(
	ctx context.Context,
	pCtx Context,
	steps []Step,
	workDir string,
) Result {
	// Important: Make a shallow copy of the PromotionContext with a deep copy of
	// the StepExecutionMetadata. We'll be modifying StepExecutionMetadata
	// in-place throughout this method, but we don't want to modify the original
	// StepExecutionMetadata that we were passed in the PromotionContext. If we
	// did, the status patch operation that will eventually be performed wouldn't
	// see any difference between the original and the modified
	// StepExecutionMetadata.
	promoCtx := pCtx
	promoCtx.StepExecutionMetadata = promoCtx.StepExecutionMetadata.DeepCopy()

	// Initialize the state which will be passed to each step.
	// This is the state that will be updated by each step,
	// and returned as the final state after all steps have
	// run.
	state := promoCtx.State.DeepCopy()
	if state == nil {
		state = make(promotion.State)
	}

	var (
		healthChecks []health.Criteria
		err          error
	)

	// Execute each step in sequence, starting from the step index specified in
	// the Context if provided.
stepLoop:
	for i := promoCtx.StartFromStep; i < int64(len(steps)); i++ {
		step := steps[i]

		// If we don't have metadata for this step yet, create it.
		if int64(len(promoCtx.StepExecutionMetadata)) == i {
			promoCtx.StepExecutionMetadata = append(
				promoCtx.StepExecutionMetadata,
				kargoapi.StepExecutionMetadata{
					Alias:           step.Alias,
					ContinueOnError: step.ContinueOnError,
				},
			)
		}
		stepExecMeta := &promoCtx.StepExecutionMetadata[i]

		select {
		case <-ctx.Done():
			stepExecMeta.Status = kargoapi.PromotionStepStatusErrored
			stepExecMeta.Message = ctx.Err().Error()
			if stepExecMeta.StartedAt != nil {
				stepExecMeta.FinishedAt = ptr.To(metav1.Now())
			}
			break stepLoop
		default:
		}

		// Shared cache for expression functions that consult the Kubernetes API.
		// By using a shared cache, we avoid repeated API calls for multiple
		// expressions that require the same data (e.g. `secret('foo').bar` and
		// `secret('foo').baz`).
		var exprDataCache *gocache.Cache
		if e.cacheFunc != nil {
			exprDataCache = e.cacheFunc()
		}

		// Check if the step should be skipped.
		var skip bool
		if skip, err = step.Skip(ctx, e.kargoClient, exprDataCache, promoCtx, state); err != nil {
			stepExecMeta.Status = kargoapi.PromotionStepStatusErrored
			stepExecMeta.Message = fmt.Sprintf("error checking if step %q should be skipped: %s", step.Alias, err)
			// Continue, because despite this failure, some steps' "if" conditions may
			// still allow them to run.
			continue
		} else if skip {
			stepExecMeta.Status = kargoapi.PromotionStepStatusSkipped
			continue // Move on to the next step
		}

		// Get the StepRunner for the step.
		runner := e.registry.getStepRunner(step.Kind)
		if runner == nil {
			stepExecMeta.Status = kargoapi.PromotionStepStatusErrored
			stepExecMeta.Message = fmt.Sprintf("no promotion step runner found for kind %q", step.Kind)
			// Continue, because despite this failure, some steps' "if" conditions may
			// still allow them to run.
			continue
		}

		// Execute the step
		if stepExecMeta.StartedAt == nil {
			stepExecMeta.StartedAt = ptr.To(metav1.Now())
		}
		result, err := e.executeStep(ctx, exprDataCache, promoCtx, step, runner, workDir, state)
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
			stepExecMeta.Status = kargoapi.PromotionStepStatusErrored
			stepExecMeta.Message = fmt.Sprintf("step %q returned an invalid status: %s", step.Alias, result.Status)
			// Continue, because despite this failure, some steps' "if" conditions may
			// still allow them to run.
			continue
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
			err = fmt.Errorf("step %q errored: %s", step.Alias, message)
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
			stepExecMeta.Status = kargoapi.PromotionStepStatusErrored
			stepExecMeta.Message = fmt.Sprintf("an unrecoverable error occurred: %s", err)
			// Continue, because despite this failure, some steps' "if" conditions may
			// still allow them to run.
			continue
		case err != nil:
			// If we get to here, the error is POTENTIALLY recoverable.
			stepExecMeta.ErrorCount++
			// Check if the error threshold has been met.
			errorThreshold := step.GetErrorThreshold(runner)
			if stepExecMeta.ErrorCount >= errorThreshold {
				// The error threshold has been met.
				stepExecMeta.FinishedAt = ptr.To(metav1.Now())
				stepExecMeta.Status = kargoapi.PromotionStepStatusErrored
				stepExecMeta.Message = fmt.Sprintf(
					"step %q met error threshold of %d: %s", step.Alias,
					errorThreshold, stepExecMeta.Message,
				)
				// Continue, because despite this failure, some steps' "if" conditions
				// may still allow them to run.
				continue
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
			stepExecMeta.Status = kargoapi.PromotionStepStatusErrored
			stepExecMeta.Message = fmt.Sprintf("step %q timed out after %s", step.Alias, timeout.String())
			// Continue, because despite this failure, some steps' "if" conditions may
			// still allow them to run.
			continue
		}

		if err != nil {
			// Treat Errored/Failed as if the step is still running so that the
			// Promotion will be requeued. The step will be retried on the next
			// reconciliation.
			stepExecMeta.Message += "; step will be retried"
			return Result{
				Status:                kargoapi.PromotionPhaseRunning,
				CurrentStep:           i,
				StepExecutionMetadata: promoCtx.StepExecutionMetadata,
				State:                 state,
				HealthChecks:          healthChecks,
			}
		}

		// If we get to here, the step is still Running (waiting for some external
		// condition to be met).
		stepExecMeta.ErrorCount = 0 // Reset the error count
		return Result{
			Status:                kargoapi.PromotionPhaseRunning,
			CurrentStep:           i,
			StepExecutionMetadata: promoCtx.StepExecutionMetadata,
			State:                 state,
			HealthChecks:          healthChecks,
		}
	}

	status, msg := determinePromoPhase(steps, promoCtx.StepExecutionMetadata)

	// All steps have succeeded, return the final state.
	return Result{
		Status:                status,
		Message:               msg,
		CurrentStep:           int64(len(steps)) - 1,
		StepExecutionMetadata: promoCtx.StepExecutionMetadata,
		State:                 state,
		HealthChecks:          healthChecks,
	}
}

// determinePromoPhase determines the final PromotionPhase as a function of the
// step configuration and step execution metadata.
func determinePromoPhase(
	steps []Step,
	stepExecMetas kargoapi.StepExecutionMetadataList,
) (kargoapi.PromotionPhase, string) {
	worstStepStatus := kargoapi.PromotionStepStatusSucceeded
	var worstMsg string
	for i, stepExecMeta := range stepExecMetas {
		if steps[i].ContinueOnError {
			// If continueOnError is set, we don't don't permit this step's outcome
			// to affect the overall PromotionPhase.
			continue
		}
		if stepExecMeta.Status.Compare(worstStepStatus) > 0 {
			worstStepStatus = stepExecMeta.Status
			worstMsg = stepExecMeta.Message
		}
	}
	switch worstStepStatus {
	case kargoapi.PromotionStepStatusSucceeded, kargoapi.PromotionStepStatusSkipped:
		return kargoapi.PromotionPhaseSucceeded, worstMsg
	case kargoapi.PromotionStepStatusAborted:
		return kargoapi.PromotionPhaseAborted, worstMsg
	case kargoapi.PromotionStepStatusFailed:
		return kargoapi.PromotionPhaseFailed, worstMsg
	case kargoapi.PromotionStepStatusErrored:
		return kargoapi.PromotionPhaseErrored, worstMsg
	default:
		// This really shouldn't ever happen. We'll treat it as an error.
		return kargoapi.PromotionPhaseErrored, worstMsg
	}
}

// executeStep executes a single Step.
func (e *simpleEngine) executeStep(
	ctx context.Context,
	cache *gocache.Cache,
	promoCtx Context,
	step Step,
	runner promotion.StepRunner,
	workDir string,
	state promotion.State,
) (promotion.StepResult, error) {
	stepCtx, err := e.prepareStepContext(ctx, cache, promoCtx, step, workDir, state)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusErrored,
		}, &promotion.TerminalError{Err: err}
	}

	result, err := runner.Run(ctx, stepCtx)
	if err != nil {
		err = fmt.Errorf("error running step %q: %w", step.Alias, err)
	}
	return result, err
}

// prepareStepContext prepares a StepContext corresponding to the provided Step.
func (e *simpleEngine) prepareStepContext(
	ctx context.Context,
	cache *gocache.Cache,
	promoCtx Context,
	step Step,
	workDir string,
	state promotion.State,
) (*promotion.StepContext, error) {
	stateCopy := state.DeepCopy()

	stepCfg, err := step.GetConfig(ctx, e.kargoClient, cache, promoCtx, stateCopy)
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
			kargoapi.CredentialTypeLabelKey: kargoapi.CredentialTypeLabelGeneric,
		},
	); err != nil {
		return nil, fmt.Errorf("error listing Secrets for Project %q: %w", project, err)
	}

	secretsMap := make(map[string]map[string]string, len(secretList.Items))
	for _, secret := range secretList.Items {
		secretsMap[secret.Name] = make(map[string]string, len(secret.Data))
		for key, value := range secret.Data {
			secretsMap[secret.Name][key] = string(value)
		}
	}
	return secretsMap, nil
}
