package promotion

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestDetermineFinalPhase(t *testing.T) {
	tests := []struct {
		name          string
		steps         []Step
		stepExecMetas kargoapi.StepExecutionMetadataList
		expectedPhase kargoapi.PromotionPhase
	}{
		{
			name:  "success",
			steps: []Step{{}},
			stepExecMetas: kargoapi.StepExecutionMetadataList{{
				Status: kargoapi.PromotionStepStatusSucceeded,
			}},
			expectedPhase: kargoapi.PromotionPhaseSucceeded,
		},
		{
			name:  "worst step result was skipped",
			steps: []Step{{}, {}},
			stepExecMetas: kargoapi.StepExecutionMetadataList{
				{
					Status: kargoapi.PromotionStepStatusSucceeded,
				},
				{
					Status: kargoapi.PromotionStepStatusSkipped,
				},
			},
			expectedPhase: kargoapi.PromotionPhaseSucceeded,
		},
		{
			name:  "worst step result was aborted",
			steps: []Step{{}, {}, {}},
			stepExecMetas: kargoapi.StepExecutionMetadataList{
				{
					Status: kargoapi.PromotionStepStatusSucceeded,
				},
				{
					Status: kargoapi.PromotionStepStatusSkipped,
				},
				{
					Status: kargoapi.PromotionStepStatusAborted,
				},
			},
			expectedPhase: kargoapi.PromotionPhaseAborted,
		},
		{
			name:  "worst step result was failed",
			steps: []Step{{}, {}, {}, {}},
			stepExecMetas: kargoapi.StepExecutionMetadataList{
				{
					Status: kargoapi.PromotionStepStatusSucceeded,
				},
				{
					Status: kargoapi.PromotionStepStatusSkipped,
				},
				{
					Status: kargoapi.PromotionStepStatusAborted,
				},
				{
					Status: kargoapi.PromotionStepStatusFailed,
				},
			},
			expectedPhase: kargoapi.PromotionPhaseFailed,
		},
		{
			name:  "worst step result was errored",
			steps: []Step{{}, {}, {}, {}, {}},
			stepExecMetas: kargoapi.StepExecutionMetadataList{
				{
					Status: kargoapi.PromotionStepStatusSucceeded,
				},
				{
					Status: kargoapi.PromotionStepStatusSkipped,
				},
				{
					Status: kargoapi.PromotionStepStatusAborted,
				},
				{
					Status: kargoapi.PromotionStepStatusFailed,
				},
				{
					Status: kargoapi.PromotionStepStatusErrored,
				},
			},
			expectedPhase: kargoapi.PromotionPhaseErrored,
		},
		{
			name: "worst step result was running",
			// This is a case that should never occur, but the logic does map this
			// to an errored phase.
			steps: []Step{{}, {}, {}},
			stepExecMetas: kargoapi.StepExecutionMetadataList{
				{
					Status: kargoapi.PromotionStepStatusSucceeded,
				},
				{
					Status: kargoapi.PromotionStepStatusSkipped,
				},
				{
					Status: kargoapi.PromotionStepStatusRunning,
				},
			},
			expectedPhase: kargoapi.PromotionPhaseErrored,
		},
		{
			name:  "step with continueOnError does not affect phase",
			steps: []Step{{}, {ContinueOnError: true}, {}},
			stepExecMetas: kargoapi.StepExecutionMetadataList{
				{
					Status: kargoapi.PromotionStepStatusSucceeded,
				},
				{
					Status: kargoapi.PromotionStepStatusErrored,
				},
				{
					Status: kargoapi.PromotionStepStatusSucceeded,
				},
			},
			expectedPhase: kargoapi.PromotionPhaseSucceeded,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase, _ := DetermineFinalPhase(tt.steps, tt.stepExecMetas)
			require.Equal(t, tt.expectedPhase, phase)
		})
	}
}
