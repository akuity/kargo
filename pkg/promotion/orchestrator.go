package promotion

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Orchestrator is an interface that defines the methods required to execute
// a series of steps in a Promotion. It is responsible for orchestrating the
// execution of steps, handling their results, and managing the state of the
// Promotion as it progresses through the steps.
type Orchestrator interface {
	// ExecuteSteps executes the provided steps in the context of the given
	// Promotion Context. It returns a Result that contains the status of the
	// Promotion after executing the steps, including any health checks that
	// were performed during the execution.
	ExecuteSteps(ctx context.Context, promoCtx Context, steps []Step) (Result, error)
}

// DetermineFinalPhase determines the final PromotionPhase based on the
// statuses of individual steps. It takes into account the ContinueOnError
// flag for each step to decide whether a step's failure should impact the
// overall PromotionPhase.
//
// It should not be used for determining the phase of an ongoing promotion
// (which would always have the phase "Running"). Instead, it is intended
// for use after all steps have completed to determine the final phase of
// the promotion.
func DetermineFinalPhase(
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
