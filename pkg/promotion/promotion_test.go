package promotion

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestContext_SetCurrentStep(t *testing.T) {
	tests := []struct {
		name       string
		context    *Context
		step       Step
		assertions func(t *testing.T, ctx *Context, result *StepMetadata)
	}{
		{
			name: "sets current step with new metadata",
			context: &Context{
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{},
			},
			step: Step{
				Alias:           "new-step",
				ContinueOnError: true,
			},
			assertions: func(t *testing.T, ctx *Context, result *StepMetadata) {
				assert.NotNil(t, result)
				assert.Equal(t, "new-step", result.Alias)
				assert.True(t, result.ContinueOnError)
				assert.Equal(t, result, ctx.GetCurrentStep())
				assert.Len(t, ctx.StepExecutionMetadata, 1)
			},
		},
		{
			name: "sets current step with existing metadata",
			context: &Context{
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{
					{
						Alias:           "existing-step",
						ContinueOnError: false,
						Status:          kargoapi.PromotionStepStatusRunning,
					},
				},
			},
			step: Step{
				Alias:           "existing-step",
				ContinueOnError: true,
			},
			assertions: func(t *testing.T, ctx *Context, result *StepMetadata) {
				assert.NotNil(t, result)
				assert.Equal(t, "existing-step", result.Alias)
				assert.False(t, result.ContinueOnError) // Should preserve existing value
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, result.Status)
				assert.Equal(t, result, ctx.GetCurrentStep())
				assert.Len(t, ctx.StepExecutionMetadata, 1)
			},
		},
		{
			name: "replaces previous current step",
			context: &Context{
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{
					{
						Alias:           "step1",
						ContinueOnError: false,
					},
					{
						Alias:           "step2",
						ContinueOnError: true,
					},
				},
			},
			step: Step{
				Alias:           "step2",
				ContinueOnError: false,
			},
			assertions: func(t *testing.T, ctx *Context, result *StepMetadata) {
				assert.NotNil(t, result)
				assert.Equal(t, "step2", result.Alias)
				assert.True(t, result.ContinueOnError) // Should preserve existing value
				assert.Equal(t, result, ctx.GetCurrentStep())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.context.SetCurrentStep(tt.step)
			tt.assertions(t, tt.context, result)
		})
	}
}

func TestContext_GetCurrentStep(t *testing.T) {
	tests := []struct {
		name       string
		context    *Context
		assertions func(t *testing.T, result *StepMetadata)
	}{
		{
			name: "returns nil when no current step is set",
			context: &Context{
				currentStepMetadata: nil,
			},
			assertions: func(t *testing.T, result *StepMetadata) {
				assert.Nil(t, result)
			},
		},
		{
			name: "returns current step metadata when set",
			context: &Context{
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{
					{
						Alias:           "current-step",
						ContinueOnError: true,
						Status:          kargoapi.PromotionStepStatusRunning,
					},
				},
			},
			assertions: func(t *testing.T, result *StepMetadata) {
				assert.NotNil(t, result)
				assert.Equal(t, "current-step", result.Alias)
				assert.True(t, result.ContinueOnError)
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, result.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set current step if we have metadata
			if len(tt.context.StepExecutionMetadata) > 0 {
				step := Step{
					Alias:           tt.context.StepExecutionMetadata[0].Alias,
					ContinueOnError: tt.context.StepExecutionMetadata[0].ContinueOnError,
				}
				tt.context.SetCurrentStep(step)
			}

			result := tt.context.GetCurrentStep()
			tt.assertions(t, result)
		})
	}
}

func TestStepMetadata_WithStatus(t *testing.T) {
	tests := []struct {
		name       string
		metadata   *StepMetadata
		status     kargoapi.PromotionStepStatus
		assertions func(t *testing.T, result *StepMetadata)
	}{
		{
			name: "sets status on empty metadata",
			metadata: &StepMetadata{
				Alias: "test-step",
			},
			status: kargoapi.PromotionStepStatusSucceeded,
			assertions: func(t *testing.T, result *StepMetadata) {
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)
				assert.Equal(t, "test-step", result.Alias)
				assert.Equal(t, result, result) // Should return self for chaining
			},
		},
		{
			name: "overwrites existing status",
			metadata: &StepMetadata{
				Alias:  "test-step",
				Status: kargoapi.PromotionStepStatusRunning,
			},
			status: kargoapi.PromotionStepStatusFailed,
			assertions: func(t *testing.T, result *StepMetadata) {
				assert.Equal(t, kargoapi.PromotionStepStatusFailed, result.Status)
				assert.Equal(t, "test-step", result.Alias)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metadata.WithStatus(tt.status)
			tt.assertions(t, result)
		})
	}
}

func TestStepMetadata_WithMessage(t *testing.T) {
	tests := []struct {
		name       string
		metadata   *StepMetadata
		message    string
		assertions func(t *testing.T, result *StepMetadata)
	}{
		{
			name: "sets message on empty metadata",
			metadata: &StepMetadata{
				Alias: "test-step",
			},
			message: "Test message",
			assertions: func(t *testing.T, result *StepMetadata) {
				assert.Equal(t, "Test message", result.Message)
				assert.Equal(t, "test-step", result.Alias)
				assert.Equal(t, result, result) // Should return self for chaining
			},
		},
		{
			name: "overwrites existing message",
			metadata: &StepMetadata{
				Alias:   "test-step",
				Message: "Old message",
			},
			message: "New message",
			assertions: func(t *testing.T, result *StepMetadata) {
				assert.Equal(t, "New message", result.Message)
				assert.Equal(t, "test-step", result.Alias)
			},
		},
		{
			name: "sets empty message",
			metadata: &StepMetadata{
				Alias:   "test-step",
				Message: "Existing message",
			},
			message: "",
			assertions: func(t *testing.T, result *StepMetadata) {
				assert.Equal(t, "", result.Message)
				assert.Equal(t, "test-step", result.Alias)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metadata.WithMessage(tt.message)
			tt.assertions(t, result)
		})
	}
}

func TestStepMetadata_WithMessagef(t *testing.T) {
	tests := []struct {
		name       string
		metadata   *StepMetadata
		format     string
		args       []any
		assertions func(t *testing.T, result *StepMetadata)
	}{
		{
			name: "formats message with single argument",
			metadata: &StepMetadata{
				Alias: "test-step",
			},
			format: "Step %s completed",
			args:   []any{"test-step"},
			assertions: func(t *testing.T, result *StepMetadata) {
				assert.Equal(t, "Step test-step completed", result.Message)
				assert.Equal(t, "test-step", result.Alias)
				assert.Equal(t, result, result) // Should return self for chaining
			},
		},
		{
			name: "formats message with multiple arguments",
			metadata: &StepMetadata{
				Alias: "test-step",
			},
			format: "Step %s failed with error %d: %s",
			args:   []any{"deploy", 500, "Internal Server Error"},
			assertions: func(t *testing.T, result *StepMetadata) {
				assert.Equal(t, "Step deploy failed with error 500: Internal Server Error", result.Message)
			},
		},
		{
			name: "formats message with no arguments",
			metadata: &StepMetadata{
				Alias: "test-step",
			},
			format: "Static message",
			args:   []any{},
			assertions: func(t *testing.T, result *StepMetadata) {
				assert.Equal(t, "Static message", result.Message)
			},
		},
		{
			name: "overwrites existing message",
			metadata: &StepMetadata{
				Alias:   "test-step",
				Message: "Old message",
			},
			format: "New formatted message: %v",
			args:   []any{42},
			assertions: func(t *testing.T, result *StepMetadata) {
				assert.Equal(t, "New formatted message: 42", result.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metadata.WithMessagef(tt.format, tt.args...)
			tt.assertions(t, result)
		})
	}
}

func TestStepMetadata_Error(t *testing.T) {
	tests := []struct {
		name       string
		metadata   *StepMetadata
		assertions func(t *testing.T, result *StepMetadata)
	}{
		{
			name: "increments error count from zero",
			metadata: &StepMetadata{
				Alias:      "test-step",
				ErrorCount: 0,
			},
			assertions: func(t *testing.T, result *StepMetadata) {
				assert.Equal(t, uint32(1), result.ErrorCount)
				assert.Equal(t, "test-step", result.Alias)
				assert.Equal(t, result, result) // Should return self for chaining
			},
		},
		{
			name: "increments existing error count",
			metadata: &StepMetadata{
				Alias:      "test-step",
				ErrorCount: 3,
			},
			assertions: func(t *testing.T, result *StepMetadata) {
				assert.Equal(t, uint32(4), result.ErrorCount)
				assert.Equal(t, "test-step", result.Alias)
			},
		},
		{
			name: "multiple error calls accumulate",
			metadata: &StepMetadata{
				Alias:      "test-step",
				ErrorCount: 1,
			},
			assertions: func(t *testing.T, result *StepMetadata) {
				// Call Error multiple times
				result.Error().Error()
				assert.Equal(t, uint32(4), result.ErrorCount) // 1 + 1 + 1 + 1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metadata.Error()
			tt.assertions(t, result)
		})
	}
}

func TestStepMetadata_Started(t *testing.T) {
	tests := []struct {
		name       string
		metadata   *StepMetadata
		assertions func(t *testing.T, result *StepMetadata)
	}{
		{
			name: "sets StartedAt when nil",
			metadata: &StepMetadata{
				Alias:     "test-step",
				StartedAt: nil,
			},
			assertions: func(t *testing.T, result *StepMetadata) {
				assert.NotNil(t, result.StartedAt)
				assert.WithinDuration(t, time.Now(), result.StartedAt.Time, time.Second)
				assert.Equal(t, uint32(0), result.ErrorCount)
				assert.Equal(t, "test-step", result.Alias)
				assert.Equal(t, result, result) // Should return self for chaining
			},
		},
		{
			name: "does not overwrite existing StartedAt",
			metadata: &StepMetadata{
				Alias:      "test-step",
				StartedAt:  ptr.To(metav1.NewTime(time.Now().Add(-time.Hour))),
				ErrorCount: 5,
			},
			assertions: func(t *testing.T, result *StepMetadata) {
				// Should preserve original time and error count
				assert.True(t, result.StartedAt.Before(ptr.To(metav1.Time{Time: time.Now().Add(-time.Minute * 50)})))
				assert.Equal(t, uint32(5), result.ErrorCount) // Should not reset error count
			},
		},
		{
			name: "resets error count when StartedAt is nil",
			metadata: &StepMetadata{
				Alias:      "test-step",
				StartedAt:  nil,
				ErrorCount: 10,
			},
			assertions: func(t *testing.T, result *StepMetadata) {
				assert.NotNil(t, result.StartedAt)
				assert.Equal(t, uint32(0), result.ErrorCount)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metadata.Started()
			tt.assertions(t, result)
		})
	}
}

func TestStepMetadata_Finished(t *testing.T) {
	tests := []struct {
		name       string
		metadata   *StepMetadata
		assertions func(t *testing.T, result *StepMetadata)
	}{
		{
			name: "sets FinishedAt when nil",
			metadata: &StepMetadata{
				Alias:      "test-step",
				FinishedAt: nil,
			},
			assertions: func(t *testing.T, result *StepMetadata) {
				assert.NotNil(t, result.FinishedAt)
				assert.WithinDuration(t, time.Now(), result.FinishedAt.Time, time.Second)
				assert.Equal(t, "test-step", result.Alias)
				assert.Equal(t, result, result) // Should return self for chaining
			},
		},
		{
			name: "does not overwrite existing FinishedAt",
			metadata: &StepMetadata{
				Alias:      "test-step",
				FinishedAt: ptr.To(metav1.NewTime(time.Now().Add(-time.Hour))),
			},
			assertions: func(t *testing.T, result *StepMetadata) {
				// Should preserve original time
				assert.True(t, result.FinishedAt.Before(ptr.To(metav1.Time{Time: time.Now().Add(-time.Minute * 50)})))
			},
		},
		{
			name: "preserves other fields",
			metadata: &StepMetadata{
				Alias:      "test-step",
				FinishedAt: nil,
				StartedAt:  ptr.To(metav1.NewTime(time.Now().Add(-time.Minute * 5))),
				ErrorCount: 2,
				Status:     kargoapi.PromotionStepStatusSucceeded,
				Message:    "Test message",
			},
			assertions: func(t *testing.T, result *StepMetadata) {
				assert.NotNil(t, result.FinishedAt)
				assert.NotNil(t, result.StartedAt)
				assert.Equal(t, uint32(2), result.ErrorCount)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)
				assert.Equal(t, "Test message", result.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metadata.Finished()
			tt.assertions(t, result)
		})
	}
}

func TestStepMetadata_ChainedCalls(t *testing.T) {
	tests := []struct {
		name       string
		metadata   *StepMetadata
		assertions func(t *testing.T, result *StepMetadata)
	}{
		{
			name: "methods can be chained together",
			metadata: &StepMetadata{
				Alias: "test-step",
			},
			assertions: func(t *testing.T, result *StepMetadata) {
				// Test method chaining
				final := result.Started().
					WithStatus(kargoapi.PromotionStepStatusRunning).
					WithMessage("Step is running").
					Error().
					Error().
					WithMessagef("Step failed after %d errors", 2).
					WithStatus(kargoapi.PromotionStepStatusFailed).
					Finished()

				assert.NotNil(t, final.StartedAt)
				assert.NotNil(t, final.FinishedAt)
				assert.Equal(t, kargoapi.PromotionStepStatusFailed, final.Status)
				assert.Equal(t, "Step failed after 2 errors", final.Message)
				assert.Equal(t, uint32(2), final.ErrorCount)
				assert.Equal(t, "test-step", final.Alias)

				// Verify all methods return the same instance
				assert.Equal(t, result, final)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assertions(t, tt.metadata)
		})
	}
}

func TestContext_GetCurrentStepIndex(t *testing.T) {
	tests := []struct {
		name              string
		context           *Context
		expectedStepIndex int64
	}{
		{
			name: "empty metadata returns 0",
			context: &Context{
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{},
			},
			expectedStepIndex: 0,
		},
		{
			name: "single step metadata returns 0",
			context: &Context{
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{
					{Alias: "step1"},
				},
			},
			expectedStepIndex: 0,
		},
		{
			name: "two steps metadata returns 1",
			context: &Context{
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{
					{Alias: "step1"},
					{Alias: "step2"},
				},
			},
			expectedStepIndex: 1,
		},
		{
			name: "three steps metadata returns 2",
			context: &Context{
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{
					{Alias: "step1"},
					{Alias: "step2"},
					{Alias: "step3"},
				},
			},
			expectedStepIndex: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.context.GetCurrentStepIndex()
			assert.Equal(t, tt.expectedStepIndex, result)
		})
	}
}

func TestContext_GetStepExecutionMetadata(t *testing.T) {
	tests := []struct {
		name       string
		context    *Context
		step       Step
		assertions func(t *testing.T, ctx *Context, result *kargoapi.StepExecutionMetadata)
	}{
		{
			name: "returns existing metadata when found",
			context: &Context{
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{
					{
						Alias:           "existing-step",
						ContinueOnError: true,
					},
				},
			},
			step: Step{
				Alias:           "existing-step",
				ContinueOnError: false,
			},
			assertions: func(t *testing.T, ctx *Context, result *kargoapi.StepExecutionMetadata) {
				assert.Equal(t, "existing-step", result.Alias)
				assert.True(t, result.ContinueOnError) // Should preserve existing value
				assert.Len(t, ctx.StepExecutionMetadata, 1)
			},
		},
		{
			name: "creates new metadata when not found",
			context: &Context{
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{
					{
						Alias:           "other-step",
						ContinueOnError: false,
					},
				},
			},
			step: Step{
				Alias:           "new-step",
				ContinueOnError: true,
			},
			assertions: func(t *testing.T, ctx *Context, result *kargoapi.StepExecutionMetadata) {
				assert.Equal(t, "new-step", result.Alias)
				assert.True(t, result.ContinueOnError)
				assert.Len(t, ctx.StepExecutionMetadata, 2)
				assert.Equal(t, "new-step", ctx.StepExecutionMetadata[1].Alias)
			},
		},
		{
			name: "creates new metadata when list is empty",
			context: &Context{
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{},
			},
			step: Step{
				Alias:           "first-step",
				ContinueOnError: false,
			},
			assertions: func(t *testing.T, ctx *Context, result *kargoapi.StepExecutionMetadata) {
				assert.Equal(t, "first-step", result.Alias)
				assert.False(t, result.ContinueOnError)
				assert.Len(t, ctx.StepExecutionMetadata, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.context.GetStepExecutionMetadata(tt.step)
			tt.assertions(t, tt.context, result)
		})
	}
}

func TestContext_DeepCopy(t *testing.T) {
	tests := []struct {
		name       string
		context    *Context
		assertions func(t *testing.T, original *Context, deepCopy Context)
	}{
		{
			name: "creates deep copy of all fields",
			context: &Context{
				UIBaseURL:     "https://example.com",
				WorkDir:       "/tmp/work",
				Project:       "test-project",
				Stage:         "test-stage",
				Promotion:     "test-promotion",
				StartFromStep: 2,
				Actor:         "test-actor",
				FreightRequests: []kargoapi.FreightRequest{
					{
						Origin: kargoapi.FreightOrigin{
							Kind: "Warehouse",
							Name: "test-warehouse",
						},
					},
				},
				Freight: kargoapi.FreightCollection{
					ID: "test-collection-id",
					Freight: map[string]kargoapi.FreightReference{
						"warehouse-1": {
							Name: "freight-1",
							Origin: kargoapi.FreightOrigin{
								Kind: "Warehouse",
								Name: "warehouse1",
							},
						},
						"warehouse-2": {
							Name: "freight-2",
							Origin: kargoapi.FreightOrigin{
								Kind: "Warehouse",
								Name: "warehouse-2",
							},
						},
					},
					VerificationHistory: []kargoapi.VerificationInfo{
						{
							ID:      "verification-1",
							Phase:   kargoapi.VerificationPhaseSuccessful,
							Message: "Test verification",
						},
					},
				},
				TargetFreightRef: kargoapi.FreightReference{
					Name: "target-freight",
					Origin: kargoapi.FreightOrigin{
						Kind: "Warehouse",
						Name: "target-warehouse",
					},
				},
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{
					{
						Alias:           "test-step",
						ContinueOnError: true,
					},
				},
				State: State{
					"key": "value",
				},
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "test-var",
						Value: "test-value",
					},
				},
			},
			assertions: func(t *testing.T, original *Context, deepCopy Context) {
				// Verify all fields are copied
				assert.Equal(t, original.UIBaseURL, deepCopy.UIBaseURL)
				assert.Equal(t, original.WorkDir, deepCopy.WorkDir)
				assert.Equal(t, original.Project, deepCopy.Project)
				assert.Equal(t, original.Stage, deepCopy.Stage)
				assert.Equal(t, original.Promotion, deepCopy.Promotion)
				assert.Equal(t, original.StartFromStep, deepCopy.StartFromStep)
				assert.Equal(t, original.Actor, deepCopy.Actor)

				// Verify FreightRequests is deep copied
				assert.Equal(t, len(original.FreightRequests), len(deepCopy.FreightRequests))
				if len(original.FreightRequests) > 0 {
					assert.Equal(t, original.FreightRequests[0].Origin.Name, deepCopy.FreightRequests[0].Origin.Name)
					// Verify it's a different slice
					assert.NotSame(t, &original.FreightRequests[0], &deepCopy.FreightRequests[0])
				}

				// Verify FreightCollection is deep copied
				assert.Equal(t, original.Freight.ID, deepCopy.Freight.ID)
				assert.Equal(t, len(original.Freight.Freight), len(deepCopy.Freight.Freight))
				assert.Equal(t, len(original.Freight.VerificationHistory), len(deepCopy.Freight.VerificationHistory))

				// Verify Freight map contents
				for key, originalFreight := range original.Freight.Freight {
					copyFreight, exists := deepCopy.Freight.Freight[key]
					assert.True(t, exists, "Freight key %s should exist in copy", key)
					assert.Equal(t, originalFreight.Name, copyFreight.Name)
					assert.Equal(t, originalFreight.Origin, copyFreight.Origin)
				}

				// Verify VerificationHistory contents
				if len(original.Freight.VerificationHistory) > 0 {
					assert.Equal(
						t,
						original.Freight.VerificationHistory[0].ID,
						deepCopy.Freight.VerificationHistory[0].ID,
					)
					assert.Equal(
						t,
						original.Freight.VerificationHistory[0].Phase,
						deepCopy.Freight.VerificationHistory[0].Phase,
					)
					assert.Equal(
						t,
						original.Freight.VerificationHistory[0].Message,
						deepCopy.Freight.VerificationHistory[0].Message,
					)
				}

				// Verify TargetFreightRef is deep copied
				assert.Equal(t, original.TargetFreightRef.Name, deepCopy.TargetFreightRef.Name)
				assert.Equal(t, original.TargetFreightRef.Origin, deepCopy.TargetFreightRef.Origin)

				// Verify StepExecutionMetadata is deep copied
				assert.Equal(t, len(original.StepExecutionMetadata), len(deepCopy.StepExecutionMetadata))
				if len(original.StepExecutionMetadata) > 0 {
					assert.Equal(t, original.StepExecutionMetadata[0].Alias, deepCopy.StepExecutionMetadata[0].Alias)
				}

				// Verify State is deep copied
				assert.Equal(t, original.State["key"], deepCopy.State["key"])

				// Verify Vars is cloned
				assert.Equal(t, len(original.Vars), len(deepCopy.Vars))
				if len(original.Vars) > 0 {
					assert.Equal(t, original.Vars[0].Name, deepCopy.Vars[0].Name)
				}

				// Verify modifications to copy don't affect original
				deepCopy.UIBaseURL = "modified"
				deepCopy.StartFromStep = 999
				deepCopy.Freight.ID = "modified-id"
				if len(deepCopy.FreightRequests) > 0 {
					deepCopy.FreightRequests[0].Origin.Name = "modified"
				}
				if len(deepCopy.StepExecutionMetadata) > 0 {
					deepCopy.StepExecutionMetadata[0].Alias = "modified"
				}
				deepCopy.State["key"] = "modified"
				if len(deepCopy.Freight.Freight) > 0 {
					for key := range deepCopy.Freight.Freight {
						freight := deepCopy.Freight.Freight[key]
						freight.Name = "modified-freight"
						deepCopy.Freight.Freight[key] = freight
						break
					}
				}

				assert.Equal(t, "https://example.com", original.UIBaseURL)
				assert.Equal(t, int64(2), original.StartFromStep)
				assert.Equal(t, "test-collection-id", original.Freight.ID)
				if len(original.FreightRequests) > 0 {
					assert.Equal(t, "test-warehouse", original.FreightRequests[0].Origin.Name)
				}
				if len(original.StepExecutionMetadata) > 0 {
					assert.Equal(t, "test-step", original.StepExecutionMetadata[0].Alias)
				}
				assert.Equal(t, "value", original.State["key"])
				if len(original.Freight.Freight) > 0 {
					for _, freight := range original.Freight.Freight {
						assert.NotEqual(t, "modified-freight", freight.Name)
						break
					}
				}
			},
		},
		{
			name: "handles nil FreightRequests",
			context: &Context{
				UIBaseURL:       "https://example.com",
				FreightRequests: nil,
				Freight: kargoapi.FreightCollection{
					ID:      "empty-collection",
					Freight: nil,
				},
			},
			assertions: func(t *testing.T, original *Context, deepCopy Context) {
				assert.Nil(t, deepCopy.FreightRequests)
				assert.Equal(t, original.UIBaseURL, deepCopy.UIBaseURL)
				assert.Equal(t, original.Freight.ID, deepCopy.Freight.ID)
				assert.Nil(t, deepCopy.Freight.Freight)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assertions(t, tt.context, tt.context.DeepCopy())
		})
	}
}
