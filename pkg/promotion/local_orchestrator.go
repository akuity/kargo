package promotion

import (
	"context"
	"fmt"
	"slices"

	gocache "github.com/patrickmn/go-cache"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/health"
)

// LocalOrchestrator is an implementation of the Orchestrator interface that
// executes steps locally using the provided StepExecutor and StepRunner
// registry.
type LocalOrchestrator struct {
	executor  StepExecutor
	registry  StepRunnerRegistry
	client    client.Client
	cacheFunc ExprDataCacheFn
}

// NewLocalOrchestrator creates a new LocalOrchestrator instance with the
// provided client, step runner registry, and cache function.
func NewLocalOrchestrator(
	registry StepRunnerRegistry,
	kargoClient, argoCDClient client.Client,
	credsDB credentials.Database,
	cacheFunc ExprDataCacheFn,
) *LocalOrchestrator {
	return &LocalOrchestrator{
		executor:  NewLocalStepExecutor(registry, kargoClient, argoCDClient, credsDB),
		registry:  registry,
		client:    kargoClient,
		cacheFunc: cacheFunc,
	}
}

// ExecuteSteps executes the provided steps in the context of the given
// Promotion context. It iterates through the steps, evaluates their "if"
// conditions, and executes them if they are not skipped. It also handles
// the execution metadata for each step, including start and finish times,
// statuses, and error handling. The method returns a Result that contains
// the final status of the Promotion after executing all steps, including
// any health checks that were performed during the execution.
func (o *LocalOrchestrator) ExecuteSteps(
	ctx context.Context,
	promoCtx Context,
	steps []Step,
) (Result, error) {
	if promoCtx.State == nil {
		// Initialize the state if it is not provided.
		promoCtx.State = make(State)
	}

	var healthChecks []health.Criteria

	// Execute each step in sequence, starting from the step specified in
	// the Context if provided.
	for i := promoCtx.StartFromStep; i < int64(len(steps)); i++ {
		step := steps[i]
		meta := promoCtx.SetCurrentStep(step)

		select {
		case <-ctx.Done():
			if meta.StartedAt != nil && meta.FinishedAt == nil {
				// If we did start the step but did not finish it,
				// we should mark it as errored due to the context being
				// canceled.
				meta.WithStatus(kargoapi.PromotionStepStatusErrored).WithMessagef(
					"step %q was canceled due to context cancellation: %s",
					step.Alias, ctx.Err(),
				).Finished()
			}
			return Result{
				Status:                kargoapi.PromotionPhaseErrored,
				Message:               fmt.Sprintf("execution canceled: %s", ctx.Err()),
				CurrentStep:           i,
				StepExecutionMetadata: promoCtx.StepExecutionMetadata,
				State:                 promoCtx.State,
				HealthChecks:          healthChecks,
			}, nil
		default:
			// Continue execution if the context is still active.
		}

		processor := NewStepEvaluator(o.client, o.newCache())

		// Only evaluate the "if" conditio when the step has not yet started.
		// If the step has already started (on a previous reconciliation), we
		// should not re-evaluate whether to skip it. Re-evaluating could cause
		// a step's own Failed status from a previous attempt to incorrectly
		// trigger the skip condition.
		if meta.StartedAt == nil {
			// Evaluate the "if" condition for the step to determine if it should
			// be executed.
			skip, err := processor.ShouldSkip(ctx, promoCtx, step)
			switch {
			case err != nil:
				meta.WithStatus(kargoapi.PromotionStepStatusErrored).WithMessagef(
					"error checking if step %q should be skipped: %s", step.Alias, err,
				)
				// Skip the step, because despite this failure, some steps' "if"
				// conditions may still allow them to run.
				continue
			case skip:
				meta.WithStatus(kargoapi.PromotionStepStatusSkipped)
				continue
			}
		}

		// Get the reg for the step (for validation purposes).
		//
		// NOTE(hidde): We primarily do this to ensure we do not mark the step
		// as started if we cannot find a runner for it. In the future, we
		// should consider validating the steps existence during the creation
		// of the Promotion, or e.g. work with a typed within the executor to
		// identify the lack of a registered runner.
		reg, err := o.registry.Get(step.Kind)
		if err != nil {
			meta.WithStatus(kargoapi.PromotionStepStatusErrored).WithMessagef(
				"error getting runner for step kind %q", step.Kind,
			)
			// Continue, because despite this failure, some steps' "if" conditions may
			// still allow them to run.
			//
			// TODO(hidde): Arguably, we should return a TerminalError here. As
			// it is an obvious misconfiguration that could have been caught
			// if our validation webhook was aware of registered steps.
			continue
		}

		// Mark the step as started.
		meta.Started()

		// Build step context for the step execution.
		stepCtx, err := processor.BuildStepContext(ctx, promoCtx, step)
		if err != nil {
			meta.WithStatus(kargoapi.PromotionStepStatusErrored).WithMessagef(
				"failed to build step context: %s", err,
			)
			continue
		}

		// Execute the step.
		result, err := o.executor.ExecuteStep(ctx, StepExecutionRequest{
			Context: *stepCtx,
			Step:    step,
		})

		// Propagate the step output to the state.
		o.propagateStepOutput(promoCtx, step, reg.Metadata, result)

		// Confirm the step has a valid status.
		if !result.Status.Valid() {
			meta.WithStatus(kargoapi.PromotionStepStatusErrored).WithMessagef(
				"step %q returned an invalid status: %s", step.Alias, result.Status,
			).Finished()
			continue
		}

		// Update the step execution metadata with the result.
		err = o.reconcileResultWithMetadata(promoCtx, step, result, err)

		// Determine the completion of the step based on the metadata.
		if !o.determineStepCompletion(promoCtx, step, reg.Metadata, err) {
			// Step incomplete; return error (if any) for progressive backoff.
			return Result{
				Status:                kargoapi.PromotionPhaseRunning,
				CurrentStep:           i,
				StepExecutionMetadata: promoCtx.StepExecutionMetadata,
				State:                 promoCtx.State,
				HealthChecks:          healthChecks,
				RetryAfter:            result.RetryAfter,
			}, err
		}

		// If the step succeeded, we can add any health checks to the list.
		if meta.Status == kargoapi.PromotionStepStatusSucceeded {
			if result.HealthCheck != nil {
				healthChecks = append(healthChecks, *result.HealthCheck)
			}
		}
	}

	status, msg := DetermineFinalPhase(steps, promoCtx.StepExecutionMetadata)

	// All steps have succeeded, return the final state.
	return Result{
		Status:                status,
		Message:               msg,
		CurrentStep:           int64(len(steps)) - 1,
		StepExecutionMetadata: promoCtx.StepExecutionMetadata,
		State:                 promoCtx.State,
		HealthChecks:          healthChecks,
	}, nil
}

func (o *LocalOrchestrator) propagateStepOutput(
	promoCtx Context,
	step Step,
	stepMeta StepRunnerMetadata,
	result StepResult,
) {
	// Update the state with the output of the step.
	promoCtx.State[step.Alias] = result.Output

	// If the step instructs that the output should be propagated to the
	// task namespace, do so.
	if slices.Contains(stepMeta.RequiredCapabilities, StepCapabilityTaskOutputPropagation) {
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

func (o *LocalOrchestrator) reconcileResultWithMetadata(
	promoCtx Context,
	step Step,
	result StepResult,
	err error,
) error {
	meta := promoCtx.GetCurrentStep()

	meta.WithStatus(result.Status).WithMessage(result.Message)

	if err != nil {
		if meta.Status != kargoapi.PromotionStepStatusFailed {
			// All states other than Errored and Failed should be mutually
			// exclusive with a hard error. If we got to here, a step has
			// violated this assumption. We will prioritize the error over the
			// status and change the status to Errored.
			meta.Status = kargoapi.PromotionStepStatusErrored
		}
		meta.WithMessage(err.Error())
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

func (o *LocalOrchestrator) determineStepCompletion(
	promoCtx Context,
	step Step,
	stepMeta StepRunnerMetadata,
	err error,
) bool {
	meta := promoCtx.GetCurrentStep()

	switch {
	case meta.Status == kargoapi.PromotionStepStatusSucceeded ||
		meta.Status == kargoapi.PromotionStepStatusSkipped:
		// Note: A step that ran briefly and self-determined it should be
		// "skipped" is treated similarly to success.
		meta.Finished()
		return true
	case IsTerminal(err):
		// This is an unrecoverable error.
		meta.WithStatus(kargoapi.PromotionStepStatusErrored).WithMessagef(
			"an unrecoverable error occurred: %s", err,
		).Finished()
		return true
	case err != nil:
		// If we get to here, the error is POTENTIALLY recoverable.
		meta.Error()
		// Check if the error threshold has been met.
		errorThreshold := step.Retry.GetErrorThreshold(stepMeta.DefaultErrorThreshold)
		if meta.ErrorCount >= errorThreshold {
			// The error threshold has been met.
			meta.WithStatus(kargoapi.PromotionStepStatusErrored).WithMessagef(
				"step %q met error threshold of %d: %s", step.Alias, errorThreshold, meta.Message,
			).Finished()
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
	timeout := step.Retry.GetTimeout(stepMeta.DefaultTimeout)
	if timeout > 0 && metav1.Now().Sub(meta.StartedAt.Time) > timeout {
		// Timeout has elapsed.
		meta.WithStatus(kargoapi.PromotionStepStatusErrored).WithMessagef(
			"step %q timed out after %s", step.Alias, timeout.String(),
		).Finished()
		// Continue, because despite this failure, some steps' "if" conditions may
		// still allow them to run.
		return true
	}

	if err != nil {
		// Treat Errored/Failed as if the step is still running so that the
		// Promotion will be requeued. The step will be retried on the next
		// reconciliation.
		meta.WithMessagef("%s; step will be retried", meta.Message)
		return false
	}

	// If we get to here, the step is still Running (waiting for some external
	// condition to be met).
	return false
}

func (o *LocalOrchestrator) newCache() *gocache.Cache {
	if o.cacheFunc == nil {
		return nil
	}
	return o.cacheFunc()
}
