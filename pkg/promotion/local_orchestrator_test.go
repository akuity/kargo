package promotion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestLocalOrchestrator_ExecuteSteps(t *testing.T) {
	tests := []struct {
		name          string
		registrations []StepRunnerRegistration
		promoCtx      Context
		steps         []Step
		assertions    func(*testing.T, Result, error)
	}{
		{
			name:  "runner not found",
			steps: []Step{{Kind: "unknown-step"}},
			assertions: func(t *testing.T, result Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Contains(t, result.Message, "error getting runner for step kind")
				assert.Equal(t, int64(0), result.CurrentStep)

				require.Len(t, result.StepExecutionMetadata, 1)

				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.StepExecutionMetadata[0].Status)
				assert.Contains(t, result.StepExecutionMetadata[0].Message, "error getting runner for step kind")
				assert.Nil(t, result.StepExecutionMetadata[0].StartedAt)
				assert.Nil(t, result.StepExecutionMetadata[0].FinishedAt)
			},
		},
		{
			name:  "error determining whether to skip step",
			steps: []Step{{If: "${{ bogus() }}"}},
			assertions: func(t *testing.T, result Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Contains(t, result.Message, "error checking if step")
				assert.Contains(t, result.Message, "should be skipped")
				assert.Equal(t, int64(0), result.CurrentStep)

				require.Len(t, result.StepExecutionMetadata, 1)

				stepExecMeta := result.StepExecutionMetadata[0]
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, stepExecMeta.Status)
				assert.Contains(t, stepExecMeta.Message, "error checking if step")
				assert.Contains(t, stepExecMeta.Message, "should be skipped")
				assert.Nil(t, stepExecMeta.StartedAt)
				assert.Nil(t, stepExecMeta.FinishedAt)
			},
		},
		{
			name: "execute all steps successfully",
			steps: []Step{
				{Kind: "success-step", Alias: "step1"},
				{Kind: "success-step", Alias: "step2"},
			},
			assertions: func(t *testing.T, result Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)
				assert.Empty(t, result.Message)
				assert.Equal(t, int64(1), result.CurrentStep)

				// Verify the result contains metadata from both steps
				require.Len(t, result.StepExecutionMetadata, 2)
				for _, metadata := range result.StepExecutionMetadata {
					assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, metadata.Status)
					assert.NotNil(t, metadata.StartedAt)
					assert.NotNil(t, metadata.FinishedAt)
				}

				// Verify state contains outputs from both steps
				assert.Equal(t, State{
					"step1": map[string]any{
						"key": "value",
					},
					"step2": map[string]any{
						"key": "value",
					},
				}, result.State)
			},
		},
		{
			name: "execute the skipped step",
			steps: []Step{
				{Kind: "skipped-step", Alias: "step1"},
				{Kind: "skipped-step", Alias: "step2"},
			},
			assertions: func(t *testing.T, result Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)
				assert.Empty(t, result.Message)
				assert.Equal(t, int64(1), result.CurrentStep)

				// Verify the result contains metadata from both steps
				require.Len(t, result.StepExecutionMetadata, 2)
				for _, metadata := range result.StepExecutionMetadata {
					assert.Equal(t, kargoapi.PromotionStepStatusSkipped, metadata.Status)
					assert.NotNil(t, metadata.StartedAt)
					assert.NotNil(t, metadata.FinishedAt)
				}

				// Verify state contains output from both steps
				assert.Equal(t, State{
					"step1": map[string]any{
						"key": "value",
					},
					"step2": map[string]any{
						"key": "value",
					},
				}, result.State)
			},
		},
		{
			name: "conditional step execution",
			steps: []Step{
				{Kind: "success-step", Alias: "step1"},
				{Kind: "error-step", Alias: "step2", If: "${{ false }}"},
				{Kind: "success-step", Alias: "step3", If: "${{ true }}"},
			},
			assertions: func(t *testing.T, result Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)
				assert.Empty(t, result.Message)
				assert.Equal(t, int64(2), result.CurrentStep)

				// Verify the result contains metadata from all steps
				require.Len(t, result.StepExecutionMetadata, 3)
				for _, metadata := range result.StepExecutionMetadata {
					switch metadata.Alias {
					case "step2":
						assert.Equal(t, kargoapi.PromotionStepStatusSkipped, metadata.Status)
						assert.Nil(t, metadata.StartedAt)
						assert.Nil(t, metadata.FinishedAt)
					default:
						assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, metadata.Status)
						assert.NotNil(t, metadata.StartedAt)
						assert.NotNil(t, metadata.FinishedAt)
					}
				}

				// Verify state contains outputs from both steps
				assert.Equal(t, State{
					"step1": map[string]any{
						"key": "value",
					},
					"step3": map[string]any{
						"key": "value",
					},
				}, result.State)
			},
		},
		{
			name: "start from middle step",
			promoCtx: Context{
				StartFromStep: 1,
				// Dummy metadata for the 0 step, which must have succeeded already if
				// we're starting from step 1
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{{
					Status: kargoapi.PromotionStepStatusSucceeded,
				}},
			},
			steps: []Step{
				// This step must have already succeeded and should not be run again
				// this time.
				{Kind: "error-step", Alias: "step1"},
				// This step should be run
				{Kind: "success-step", Alias: "step2"},
			},
			assertions: func(t *testing.T, result Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)
				assert.Empty(t, result.Message)
				assert.Equal(t, int64(1), result.CurrentStep)

				// Verify the result contains metadata from both steps
				require.Len(t, result.StepExecutionMetadata, 2)
				// We're not bothering with assertions on the dummy metadata for the 0
				// step.
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.StepExecutionMetadata[1].Status)
				assert.NotNil(t, result.StepExecutionMetadata[1].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[1].FinishedAt)

				// Verify only second step output is in state
				assert.Equal(t, State{
					"step2": map[string]any{
						"key": "value",
					},
				}, result.State)
			},
		},
		{
			name: "terminal error on step execution",
			steps: []Step{
				{Kind: "success-step", Alias: "step1"},
				{Kind: "terminal-error-step", Alias: "step2"},
			},
			assertions: func(t *testing.T, result Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Contains(t, result.Message, "an unrecoverable error occurred")
				assert.Equal(t, int64(1), result.CurrentStep)

				require.Len(t, result.StepExecutionMetadata, 2)

				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.StepExecutionMetadata[0].Status)
				assert.NotNil(t, result.StepExecutionMetadata[0].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[0].FinishedAt)

				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.StepExecutionMetadata[1].Status)
				assert.NotNil(t, result.StepExecutionMetadata[1].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[1].FinishedAt)
				assert.Contains(t, result.StepExecutionMetadata[1].Message, "something went wrong")

				// Verify first step output is preserved in state
				assert.Equal(t, State{
					"step1": map[string]any{
						"key": "value",
					},
					"step2": map[string]any(nil),
				}, result.State)
			},
		},
		{
			name: "non-terminal error on step execution; error threshold met",
			steps: []Step{
				{Kind: "success-step", Alias: "step1"},
				{Kind: "error-step", Alias: "step2"},
			},
			assertions: func(t *testing.T, result Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Contains(t, result.Message, "met error threshold")
				assert.Equal(t, int64(1), result.CurrentStep)

				require.Len(t, result.StepExecutionMetadata, 2)

				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.StepExecutionMetadata[0].Status)
				assert.NotNil(t, result.StepExecutionMetadata[0].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[0].FinishedAt)

				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.StepExecutionMetadata[1].Status)
				assert.NotNil(t, result.StepExecutionMetadata[1].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[1].FinishedAt)
				assert.Contains(t, result.StepExecutionMetadata[1].Message, "something went wrong")

				// Verify first step output is preserved in state
				assert.Equal(t, State{
					"step1": map[string]any{
						"key": "value",
					},
					"step2": map[string]any(nil),
				}, result.State)
			},
		},
		{
			name: "non-terminal error on step execution; error threshold not met",
			steps: []Step{
				{
					Kind:  "error-step",
					Alias: "step1",
					Retry: &kargoapi.PromotionStepRetry{ErrorThreshold: 3},
				},
			},
			assertions: func(t *testing.T, result Result, err error) {
				require.Error(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseRunning, result.Status)
				assert.Empty(t, result.Message)
				assert.Equal(t, int64(0), result.CurrentStep)

				require.Len(t, result.StepExecutionMetadata, 1)

				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.StepExecutionMetadata[0].Status)
				assert.NotNil(t, result.StepExecutionMetadata[0].StartedAt)
				assert.Nil(t, result.StepExecutionMetadata[0].FinishedAt)
				assert.Equal(t, uint32(1), result.StepExecutionMetadata[0].ErrorCount)
				assert.Contains(t, result.StepExecutionMetadata[0].Message, "will be retried")
			},
		},
		{
			name: "non-terminal error on step execution; timeout elapsed",
			promoCtx: Context{
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{{
					// Start time is set to an hour ago
					StartedAt: ptr.To(metav1.NewTime(time.Now().Add(-time.Hour))),
				}},
			},
			steps: []Step{
				{
					Kind: "error-step",
					Retry: &kargoapi.PromotionStepRetry{
						ErrorThreshold: 3,
						Timeout: &metav1.Duration{
							Duration: time.Hour,
						},
					},
				},
			},
			assertions: func(t *testing.T, result Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Contains(t, result.Message, "timed out after")
				assert.Equal(t, int64(0), result.CurrentStep)

				require.Len(t, result.StepExecutionMetadata, 1)

				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.StepExecutionMetadata[0].Status)
				assert.NotNil(t, result.StepExecutionMetadata[0].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[0].FinishedAt)
				assert.Equal(t, uint32(1), result.StepExecutionMetadata[0].ErrorCount)
			},
		},
		{
			name: "step is still running; timeout elapsed",
			promoCtx: Context{
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{{
					// Start time is set to an hour ago
					StartedAt: ptr.To(metav1.NewTime(time.Now().Add(-time.Hour))),
				}},
			},
			steps: []Step{
				{
					Kind: "running-step",
					Retry: &kargoapi.PromotionStepRetry{
						Timeout: &metav1.Duration{
							Duration: time.Hour,
						},
					},
				},
			},
			assertions: func(t *testing.T, result Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Contains(t, result.Message, "timed out after")
				assert.Equal(t, int64(0), result.CurrentStep)

				require.Len(t, result.StepExecutionMetadata, 1)

				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.StepExecutionMetadata[0].Status)
				assert.NotNil(t, result.StepExecutionMetadata[0].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[0].FinishedAt)
			},
		},
		{
			name:  "step is still running; timeout not elapsed",
			steps: []Step{{Kind: "running-step"}},
			assertions: func(t *testing.T, result Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseRunning, result.Status)
				assert.Empty(t, result.Message)
				assert.Equal(t, int64(0), result.CurrentStep)

				require.Len(t, result.StepExecutionMetadata, 1)

				assert.Equal(t, kargoapi.PromotionStepStatusRunning, result.StepExecutionMetadata[0].Status)
				assert.NotNil(t, result.StepExecutionMetadata[0].StartedAt)
				assert.Nil(t, result.StepExecutionMetadata[0].FinishedAt)
			},
		},
		{
			name: "context cancellation",
			steps: []Step{
				{Kind: "context-waiter"}, // Closes context and errors
				{Kind: "success-step"},   // Won't run because of canceled context
			},
			assertions: func(t *testing.T, result Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Contains(t, result.Message, context.Canceled.Error())
				assert.Equal(t, int64(1), result.CurrentStep)

				require.Len(t, result.StepExecutionMetadata, 1)

				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.StepExecutionMetadata[0].Status)
				assert.Contains(t, result.StepExecutionMetadata[0].Message, context.Canceled.Error())
				assert.NotNil(t, result.StepExecutionMetadata[0].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[0].FinishedAt)
			},
		},
		{
			name: "output propagation to task namespace",
			registrations: []StepRunnerRegistration{{
				Name: "task-level-output-step",
				Value: func(_ StepRunnerCapabilities) StepRunner {
					return &MockStepRunner{
						RunResult: StepResult{
							Status: kargoapi.PromotionStepStatusSucceeded,
							Output: map[string]any{"test": "value"},
						},
					}
				},
				Metadata: StepRunnerMetadata{
					RequiredCapabilities: []StepRunnerCapability{
						StepCapabilityTaskOutputPropagation,
					},
				},
			}},
			steps: []Step{{
				Kind:  "task-level-output-step",
				Alias: "task-1::custom-output",
			}},
			assertions: func(t *testing.T, result Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)
				assert.Empty(t, result.Message)
				assert.Equal(t, int64(0), result.CurrentStep)

				require.Len(t, result.StepExecutionMetadata, 1)

				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.StepExecutionMetadata[0].Status)
				assert.NotNil(t, result.StepExecutionMetadata[0].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[0].FinishedAt)
				assert.Equal(t, State{
					// The output should be available under both the task alias
					// and the step alias.
					"task-1::custom-output": map[string]any{
						"test": "value",
					},
					"task-1": map[string]any{
						"test": "value",
					},
				}, result.State)
			},
		},
		{
			name: "stand alone output composition step",
			registrations: []StepRunnerRegistration{{
				Name: "task-level-output-step",
				Value: func(_ StepRunnerCapabilities) StepRunner {
					return &MockStepRunner{
						RunResult: StepResult{
							Status: kargoapi.PromotionStepStatusSucceeded,
							Output: map[string]any{"test": "value"},
						},
					}
				},
				Metadata: StepRunnerMetadata{
					RequiredCapabilities: []StepRunnerCapability{
						StepCapabilityTaskOutputPropagation,
					},
				},
			}},
			steps: []Step{{
				Kind:  "task-level-output-step",
				Alias: "custom-output",
			}},
			assertions: func(t *testing.T, result Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)
				assert.Empty(t, result.Message)
				assert.Equal(t, int64(0), result.CurrentStep)

				require.Len(t, result.StepExecutionMetadata, 1)

				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.StepExecutionMetadata[0].Status)
				assert.NotNil(t, result.StepExecutionMetadata[0].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[0].FinishedAt)
				assert.Equal(t, State{
					"custom-output": map[string]any{
						"test": "value",
					},
				}, result.State)
			},
		},
		{
			name: "previously failed step does not skip on retry",
			promoCtx: Context{
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{{
					Alias:     "step1",
					StartedAt: ptr.To(metav1.NewTime(time.Now().Add(-time.Minute))),
					Status:    kargoapi.PromotionStepStatusFailed,
					Message:   "previous failure",
				}},
			},
			steps: []Step{
				{Kind: "success-step", Alias: "step1"},
			},
			assertions: func(t *testing.T, result Result, err error) {
				require.NoError(t, err)

				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)
				assert.Equal(t, int64(0), result.CurrentStep)

				require.Len(t, result.StepExecutionMetadata, 1)

				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.StepExecutionMetadata[0].Status)
				assert.NotNil(t, result.StepExecutionMetadata[0].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[0].FinishedAt)
			},
		},
		{
			name: "panic during step execution",
			registrations: []StepRunnerRegistration{
				{
					Name: "success-step",
					Value: func(_ StepRunnerCapabilities) StepRunner {
						return &MockStepRunner{
							RunResult: StepResult{
								Status: kargoapi.PromotionStepStatusSucceeded,
							},
						}
					},
				},
				{
					Name: "panic-step",
					Value: func(_ StepRunnerCapabilities) StepRunner {
						return &MockStepRunner{
							RunFunc: func(_ context.Context, _ *StepContext) (StepResult, error) {
								panic("something went wrong")
							},
						}
					},
				},
			},
			steps: []Step{
				{Kind: "success-step", Alias: "step1"},
				{Kind: "panic-step", Alias: "step2"},
				{Kind: "success-step", Alias: "step3"},
			},
			assertions: func(t *testing.T, result Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Contains(t, result.Message, "something went wrong")
				assert.Equal(t, int64(2), result.CurrentStep)

				require.Len(t, result.StepExecutionMetadata, 3)

				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.StepExecutionMetadata[0].Status)
				assert.NotNil(t, result.StepExecutionMetadata[0].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[0].FinishedAt)

				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.StepExecutionMetadata[1].Status)
				assert.NotNil(t, result.StepExecutionMetadata[1].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[1].FinishedAt)
				assert.Contains(t, result.StepExecutionMetadata[1].Message, "something went wrong")

				assert.Equal(t, kargoapi.PromotionStepStatusSkipped, result.StepExecutionMetadata[2].Status)
				assert.Nil(t, result.StepExecutionMetadata[2].StartedAt)
				assert.Nil(t, result.StepExecutionMetadata[2].FinishedAt)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			registry := MustNewStepRunnerRegistry(
				StepRunnerRegistration{
					Name: "success-step",
					Value: func(_ StepRunnerCapabilities) StepRunner {
						return &MockStepRunner{
							RunResult: StepResult{
								Status: kargoapi.PromotionStepStatusSucceeded,
								Output: map[string]any{"key": "value"},
							},
						}
					},
				},
				StepRunnerRegistration{
					Name: "skipped-step",
					Value: func(_ StepRunnerCapabilities) StepRunner {
						return &MockStepRunner{
							RunResult: StepResult{
								Status: kargoapi.PromotionStepStatusSkipped,
								Output: map[string]any{"key": "value"},
							},
						}
					},
				},
				StepRunnerRegistration{
					Name: "running-step",
					Value: func(_ StepRunnerCapabilities) StepRunner {
						return &MockStepRunner{
							RunResult: StepResult{
								Status: kargoapi.PromotionStepStatusRunning,
							},
						}
					},
				},
				StepRunnerRegistration{
					Name: "error-step",
					Value: func(_ StepRunnerCapabilities) StepRunner {
						return &MockStepRunner{
							RunResult: StepResult{
								Status: kargoapi.PromotionStepStatusErrored,
							},
							RunErr: errors.New("something went wrong"),
						}
					},
				},
				StepRunnerRegistration{
					Name: "terminal-error-step",
					Value: func(_ StepRunnerCapabilities) StepRunner {
						return &MockStepRunner{
							RunResult: StepResult{
								Status: kargoapi.PromotionStepStatusErrored,
							},
							RunErr: &TerminalError{Err: errors.New("something went wrong")},
						}
					},
				},
				StepRunnerRegistration{
					Name: "context-waiter",
					Value: func(_ StepRunnerCapabilities) StepRunner {
						return &MockStepRunner{
							RunFunc: func(ctx context.Context, _ *StepContext) (StepResult, error) {
								cancel()
								<-ctx.Done()
								return StepResult{Status: kargoapi.PromotionStepStatusErrored}, ctx.Err()
							},
						}
					},
				},
			)

			for _, reg := range tt.registrations {
				registry.MustRegister(reg)
			}

			orchestrator := NewLocalOrchestrator(
				registry,
				fake.NewClientBuilder().Build(),
				fake.NewClientBuilder().Build(),
				nil,
				nil,
			)

			tt.promoCtx.WorkDir = t.TempDir()

			result, err := orchestrator.ExecuteSteps(ctx, tt.promoCtx, tt.steps)
			tt.assertions(t, result, err)
		})
	}
}
