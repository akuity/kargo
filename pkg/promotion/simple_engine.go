package promotion

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"

	gocache "github.com/patrickmn/go-cache"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/health"
)

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
	executor    StepExecutor
	registry    stepRunnerRegistry
	kargoClient client.Client
	cacheFunc   ExprDataCacheFn
}

// NewSimpleEngine returns a simple implementation of the Engine interface that
// uses built-in StepRunners.
func NewSimpleEngine(
	kargoClient client.Client,
	argocdClient client.Client,
	credsDB credentials.Database,
	cacheFunc ExprDataCacheFn,
) Engine {
	return &simpleEngine{
		executor:    NewLocalStepExecutor(stepRunnerReg, kargoClient, argocdClient, credsDB),
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
) (_ Result, err error) {
	if promoCtx.WorkDir == "" {
		// If no working directory is provided, we create a temporary one.
		if promoCtx.WorkDir, err = e.setupWorkDir(promoCtx.WorkDir); err != nil {
			return Result{Status: kargoapi.PromotionPhaseErrored}, err
		}
		defer os.RemoveAll(promoCtx.WorkDir)
	}

	result := e.executeSteps(ctx, promoCtx, steps)
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
) Result {
	// NB: Make a deep copy of the Context so that we don't modify the original.
	promoCtx := pCtx.DeepCopy()
	if promoCtx.State == nil {
		promoCtx.State = make(State)
	}

	var (
		healthChecks []health.Criteria
	)

	// Execute each step in sequence, starting from the step index specified in
	// the Context if provided.
	for i := promoCtx.StartFromStep; i < int64(len(steps)); i++ {
		step := steps[i]

		stepExecMeta := e.prepareStepMetadata(&promoCtx, step)

		if e.isContextCanceled(ctx, stepExecMeta) {
			break
		}

		// Shared cache for expression functions that consult the Kubernetes API.
		// By using a shared cache, we avoid repeated API calls for multiple
		// expressions that require the same data (e.g. `secret('foo').bar` and
		// `secret('foo').baz`).
		var exprDataCache *gocache.Cache
		if e.cacheFunc != nil {
			exprDataCache = e.cacheFunc()
		}

		// Create StepEvaluator for this step with the shared cache
		evaluator := NewStepEvaluator(e.kargoClient, exprDataCache)

		if e.shouldSkipStep(ctx, evaluator, promoCtx, step, stepExecMeta) {
			continue
		}

		// Get a StepRunner for the step.
		registration := e.registry.getStepRunnerRegistration(step.Kind)
		if registration == nil {
			stepExecMeta.Status = kargoapi.PromotionStepStatusErrored
			stepExecMeta.Message = fmt.Sprintf("no promotion step runner found for kind %q", step.Kind)
			// Continue, because despite this failure, some steps' "if" conditions may
			// still allow them to run.
			//
			// TODO(hidde): Arguably, we should return a TerminalError here. As
			// it is an obvious misconfiguration that could have been caught
			// if our validation webhook was aware of registered steps.
			continue
		}

		// Mark the step as started.
		if stepExecMeta.StartedAt == nil {
			stepExecMeta.StartedAt = ptr.To(metav1.Now())
		}

		// Build step context for execution.
		stepCtx, err := evaluator.BuildStepContext(ctx, promoCtx, step)
		if err != nil {
			stepExecMeta.Status = kargoapi.PromotionStepStatusErrored
			stepExecMeta.Message = fmt.Sprintf("error building step context: %s", err)
			stepExecMeta.FinishedAt = ptr.To(metav1.Now())
			continue
		}

		// Execute the step using the executor.
		result, err := e.executor.ExecuteStep(ctx, StepExecutionRequest{
			Context: *stepCtx,
			Step:    step,
		})

		// Propagate the output of the step to the state.
		e.propagateStepOutput(promoCtx, step, registration.Metadata, result)

		// Confirm that the step has a valid status.
		if !isValidStepStatus(result.Status) {
			stepExecMeta.FinishedAt = ptr.To(metav1.Now())
			stepExecMeta.Status = kargoapi.PromotionStepStatusErrored
			stepExecMeta.Message = fmt.Sprintf("step %q returned an invalid status: %s", step.Alias, result.Status)
			continue
		}

		// Update the step execution metadata with the result.
		err = e.reconcileResultWithMetadata(stepExecMeta, step, result, err)

		// Determine what to do based on the result.
		if !e.determineStepCompletion(step, registration.Metadata, stepExecMeta, err) {
			// The step is still running, so we need to wait
			return Result{
				Status:                kargoapi.PromotionPhaseRunning,
				CurrentStep:           i,
				StepExecutionMetadata: promoCtx.StepExecutionMetadata,
				State:                 promoCtx.State,
				HealthChecks:          healthChecks,
			}
		}

		// If the step succeeded, we can add any health checks to the list.
		if stepExecMeta.Status == kargoapi.PromotionStepStatusSucceeded {
			if result.HealthCheck != nil {
				healthChecks = append(healthChecks, *result.HealthCheck)
			}
		}
	}

	status, msg := determinePromoPhase(steps, promoCtx.StepExecutionMetadata)

	// All steps have succeeded, return the final state.
	return Result{
		Status:                status,
		Message:               msg,
		CurrentStep:           int64(len(steps)) - 1,
		StepExecutionMetadata: promoCtx.StepExecutionMetadata,
		State:                 promoCtx.State,
		HealthChecks:          healthChecks,
	}
}

func (e *simpleEngine) prepareStepMetadata(promoCtx *Context, step Step) *kargoapi.StepExecutionMetadata {
	for i := range promoCtx.StepExecutionMetadata {
		if promoCtx.StepExecutionMetadata[i].Alias == step.Alias {
			// Found existing metadata for this step, return it.
			return &promoCtx.StepExecutionMetadata[i]
		}
	}

	// If not found, append new metadata
	promoCtx.StepExecutionMetadata = append(
		promoCtx.StepExecutionMetadata,
		kargoapi.StepExecutionMetadata{
			Alias:           step.Alias,
			ContinueOnError: step.ContinueOnError,
		},
	)
	return &promoCtx.StepExecutionMetadata[len(promoCtx.StepExecutionMetadata)-1]
}

func (e *simpleEngine) isContextCanceled(ctx context.Context, meta *kargoapi.StepExecutionMetadata) bool {
	select {
	case <-ctx.Done():
		meta.Status = kargoapi.PromotionStepStatusErrored
		meta.Message = ctx.Err().Error()
		if meta.StartedAt != nil {
			meta.FinishedAt = ptr.To(metav1.Now())
		}
		return true
	default:
		return false
	}
}

func (e *simpleEngine) shouldSkipStep(
	ctx context.Context,
	evaluator *StepEvaluator,
	promoCtx Context,
	step Step,
	meta *kargoapi.StepExecutionMetadata,
) bool {
	skip, err := evaluator.ShouldSkip(ctx, promoCtx, step)
	if err != nil {
		meta.Status = kargoapi.PromotionStepStatusErrored
		meta.Message = fmt.Sprintf("error checking if step %q should be skipped: %s", step.Alias, err)
		// Skip the step, because despite this failure, some steps' "if"
		// conditions may still allow them to run.
		return true
	}

	if skip {
		meta.Status = kargoapi.PromotionStepStatusSkipped
		// Skip the step because it was explicitly skipped.
		return true
	}

	return false
}

func (e *simpleEngine) propagateStepOutput(
	promoCtx Context,
	step Step,
	stepRunnerMeta StepRunnerMetadata,
	result StepResult,
) {
	// Update the state with the output of the step.
	promoCtx.State[step.Alias] = result.Output

	// If the step instructs that the output should be propagated to the
	// task namespace, do so.
	if slices.Contains(
		stepRunnerMeta.RequiredCapabilities,
		StepCapabilityTaskOutputPropagation,
	) {
		if aliasNamespace := getAliasNamespace(step.Alias); aliasNamespace != "" {
			if promoCtx.State[aliasNamespace] == nil {
				promoCtx.State[aliasNamespace] = make(map[string]any)
			}
			for k, v := range result.Output {
				promoCtx.State[aliasNamespace].(map[string]any)[k] = v // nolint: forcetypeassert
			}
		}
	}
}

func (e *simpleEngine) reconcileResultWithMetadata(
	meta *kargoapi.StepExecutionMetadata,
	step Step,
	result StepResult,
	err error,
) error {
	meta.Status = result.Status
	meta.Message = result.Message

	if err != nil {
		if meta.Status != kargoapi.PromotionStepStatusFailed {
			// All states other than Errored and Failed should be mutually
			// exclusive with a hard error. If we got to here, a step has
			// violated this assumption. We will prioritize the error over the
			// status and change the status to Errored.
			meta.Status = kargoapi.PromotionStepStatusErrored
		}
		meta.Message = err.Error()
		return err
	}

	if result.Status == kargoapi.PromotionStepStatusErrored {
		message := meta.Message
		if message == "" {
			message = "no details provided"
		}
		// A nil err should be mutually exclusive with an Errored status. If we
		// got to here, a step has violated this assumption. We will prioritize
		// the Errored status over the nil error and create an error.
		err = fmt.Errorf("step %q errored: %s", step.Alias, message)
		return err
	}

	return nil
}

func (e *simpleEngine) determineStepCompletion(
	step Step,
	runnerMeta StepRunnerMetadata,
	execMeta *kargoapi.StepExecutionMetadata,
	err error,
) bool {
	switch {
	case execMeta.Status == kargoapi.PromotionStepStatusSucceeded ||
		execMeta.Status == kargoapi.PromotionStepStatusSkipped:
		// Note: A step that ran briefly and self-determined it should be
		// "skipped" is treated similarly to success.
		execMeta.FinishedAt = ptr.To(metav1.Now())
		return true
	case IsTerminal(err):
		// This is an unrecoverable error.
		execMeta.FinishedAt = ptr.To(metav1.Now())
		execMeta.Status = kargoapi.PromotionStepStatusErrored
		execMeta.Message = fmt.Sprintf("an unrecoverable error occurred: %s", err)
		// Continue, because despite this failure, some steps' "if" conditions may
		// still allow them to run.
		return true
	case err != nil:
		// If we get to here, the error is POTENTIALLY recoverable.
		execMeta.ErrorCount++
		// Check if the error threshold has been met.
		errorThreshold := step.Retry.GetErrorThreshold(runnerMeta.DefaultErrorThreshold)
		if execMeta.ErrorCount >= errorThreshold {
			// The error threshold has been met.
			execMeta.FinishedAt = ptr.To(metav1.Now())
			execMeta.Status = kargoapi.PromotionStepStatusErrored
			execMeta.Message = fmt.Sprintf(
				"step %q met error threshold of %d: %s", step.Alias,
				errorThreshold, execMeta.Message,
			)
			// Continue, because despite this failure, some steps' "if" conditions
			// may still allow them to run.
			return true
		}
	}

	// If we get to here, the step is either Running (waiting for some external
	// condition to be met) or it Errored/Failed but did not meet the error
	// threshold. Now we need to check if the timeout has elapsed. A nil timeout
	// or any non-positive timeout interval are treated as NO timeout, although
	// a nil timeout really shouldn't happen.
	timeout := step.Retry.GetTimeout(runnerMeta.DefaultTimeout)
	if timeout > 0 && metav1.Now().Sub(execMeta.StartedAt.Time) > timeout {
		// Timeout has elapsed.
		execMeta.FinishedAt = ptr.To(metav1.Now())
		execMeta.Status = kargoapi.PromotionStepStatusErrored
		execMeta.Message = fmt.Sprintf("step %q timed out after %s", step.Alias, timeout.String())
		// Continue, because despite this failure, some steps' "if" conditions may
		// still allow them to run.
		return true
	}

	if err != nil {
		// Treat Errored/Failed as if the step is still running so that the
		// Promotion will be requeued. The step will be retried on the next
		// reconciliation.
		execMeta.Message += "; step will be retried"
		return false
	}

	// If we get to here, the step is still Running (waiting for some external
	// condition to be met).
	execMeta.ErrorCount = 0 // Reset the error count
	return false
}

func isValidStepStatus(status kargoapi.PromotionStepStatus) bool {
	switch status {
	case kargoapi.PromotionStepStatusSucceeded,
		kargoapi.PromotionStepStatusSkipped,
		kargoapi.PromotionStepStatusAborted,
		kargoapi.PromotionStepStatusFailed,
		kargoapi.PromotionStepStatusErrored,
		kargoapi.PromotionStepStatusRunning:
		return true
	default:
		return false
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
			// If continueOnError is set, we don't permit this step's outcome
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
