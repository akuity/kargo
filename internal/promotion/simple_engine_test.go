package promotion

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
)

func TestSimpleEngine_Promote(t *testing.T) {
	tests := []struct {
		name        string
		promoCtx    Context
		steps       []Step
		interceptor interceptor.Funcs
		assertions  func(*testing.T, Result, error)
	}{
		{
			name: "successful promotion",
			promoCtx: Context{
				Project: "test-project",
				State:   promotion.State{"existing": "state"},
			},
			steps: []Step{
				{Kind: "success-step"},
			},
			assertions: func(t *testing.T, result Result, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)
				assert.NotNil(t, result.State)
				assert.Equal(t, "state", result.State["existing"])
				assert.Equal(t, int64(0), result.CurrentStep)
			},
		},
		{
			name: "failed promotion",
			promoCtx: Context{
				Project: "test-project",
			},
			steps: []Step{
				{Kind: "error-step"},
			},
			assertions: func(t *testing.T, result Result, err error) {
				assert.Error(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.ErrorContains(t, err, "step execution failed")
			},
		},
		{
			name: "context cancellation",
			promoCtx: Context{
				Project: "test-project",
			},
			steps: []Step{
				{Kind: "context-waiter"},
			},
			assertions: func(t *testing.T, result Result, err error) {
				assert.ErrorContains(t, err, "met error threshold")
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Len(t, result.StepExecutionMetadata, 1)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.StepExecutionMetadata[0].Status)
				assert.Contains(t, result.StepExecutionMetadata[0].Message, context.Canceled.Error())
			},
		},
		{
			name: "secrets retrieval failure",
			promoCtx: Context{
				Project: "test-project",
			},
			steps: []Step{
				{Kind: "success-step"},
			},
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, result Result, err error) {
				assert.ErrorContains(t, err, "error listing Secrets for Project")
				assert.ErrorContains(t, err, "something went wrong")
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			testRegistry := stepRunnerRegistry{}
			testRegistry.register(
				&mockStepRunner{
					name:      "success-step",
					runResult: promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded},
				},
			)
			testRegistry.register(
				&mockStepRunner{
					name:      "error-step",
					runResult: promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
					runErr:    errors.New("something went wrong"),
				},
			)
			testRegistry.register(
				&mockStepRunner{
					name: "context-waiter",
					runFunc: func(ctx context.Context, _ *promotion.StepContext) (promotion.StepResult, error) {
						cancel() // Cancel context immediately
						<-ctx.Done()
						return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, ctx.Err()
					},
				},
			)

			engine := &simpleEngine{
				registry: testRegistry,
				kargoClient: fake.NewClientBuilder().
					WithInterceptorFuncs(tt.interceptor).
					Build(),
			}

			result, err := engine.Promote(ctx, tt.promoCtx, tt.steps)
			tt.assertions(t, result, err)
		})
	}
}

func TestSimpleEngine_executeSteps(t *testing.T) {
	tests := []struct {
		name        string
		stepRunners []promotion.StepRunner
		promoCtx    Context
		steps       []Step
		assertions  func(*testing.T, Result, error)
	}{
		{
			name: "fail on invalid step alias",
			steps: []Step{
				{Kind: "success-step", Alias: "step-1"},
			},
			assertions: func(t *testing.T, result Result, err error) {
				assert.ErrorContains(t, err, "is forbidden")
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Equal(t, int64(0), result.CurrentStep)
			},
		},
		{
			name:  "runner not found",
			steps: []Step{{Kind: "unknown-step"}},
			assertions: func(t *testing.T, result Result, err error) {
				assert.ErrorContains(t, err, "no promotion step runner found for kind")
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Equal(t, int64(0), result.CurrentStep)
			},
		},
		{
			name: "execute all steps successfully",
			steps: []Step{
				{Kind: "success-step", Alias: "step1"},
				{Kind: "success-step", Alias: "step2"},
			},
			assertions: func(t *testing.T, result Result, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)
				assert.Equal(t, int64(1), result.CurrentStep)

				// Verify the result contains metadata from both steps
				assert.Len(t, result.StepExecutionMetadata, 2)
				for _, metadata := range result.StepExecutionMetadata {
					assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, metadata.Status)
					assert.NotNil(t, metadata.StartedAt)
					assert.NotNil(t, metadata.FinishedAt)
				}

				// Verify state contains outputs from both steps
				assert.Equal(t, promotion.State{
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
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)
				assert.Equal(t, int64(2), result.CurrentStep)

				// Verify the result contains metadata from all steps
				assert.Len(t, result.StepExecutionMetadata, 3)
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
				assert.Equal(t, promotion.State{
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
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{{}},
			},
			steps: []Step{
				// This step must have already succeeded and should not be run again
				// this time.
				{Kind: "error-step", Alias: "step1"},
				// This step should be run
				{Kind: "success-step", Alias: "step2"},
			},
			assertions: func(t *testing.T, result Result, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)
				assert.Equal(t, int64(1), result.CurrentStep)

				// Verify the result contains metadata from both steps
				assert.Len(t, result.StepExecutionMetadata, 2)
				// We're not bothering with assertions on the dummy metadata for the 0
				// step.
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.StepExecutionMetadata[1].Status)
				assert.NotNil(t, result.StepExecutionMetadata[1].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[1].FinishedAt)

				// Verify only second step output is in state
				assert.Equal(t, promotion.State{
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
				assert.ErrorContains(t, err, "an unrecoverable error occurred")
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Equal(t, int64(1), result.CurrentStep)
				assert.Len(t, result.StepExecutionMetadata, 2)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.StepExecutionMetadata[0].Status)
				assert.NotNil(t, result.StepExecutionMetadata[0].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[0].FinishedAt)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.StepExecutionMetadata[1].Status)
				assert.NotNil(t, result.StepExecutionMetadata[1].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[1].FinishedAt)
				assert.Contains(t, result.StepExecutionMetadata[1].Message, "something went wrong")

				// Verify first step output is preserved in state
				assert.Equal(t, promotion.State{
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
				assert.ErrorContains(t, err, "met error threshold")
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Equal(t, int64(1), result.CurrentStep)
				assert.Len(t, result.StepExecutionMetadata, 2)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.StepExecutionMetadata[0].Status)
				assert.NotNil(t, result.StepExecutionMetadata[0].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[0].FinishedAt)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.StepExecutionMetadata[1].Status)
				assert.NotNil(t, result.StepExecutionMetadata[1].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[1].FinishedAt)
				assert.Contains(t, result.StepExecutionMetadata[1].Message, "something went wrong")

				// Verify first step output is preserved in state
				assert.Equal(t, promotion.State{
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
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseRunning, result.Status)
				assert.Equal(t, int64(0), result.CurrentStep)
				assert.Len(t, result.StepExecutionMetadata, 1)
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
				assert.ErrorContains(t, err, "timeout")
				assert.ErrorContains(t, err, "has elapsed")
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Equal(t, int64(0), result.CurrentStep)
				assert.Len(t, result.StepExecutionMetadata, 1)
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
				assert.ErrorContains(t, err, "timeout")
				assert.ErrorContains(t, err, "has elapsed")
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Equal(t, int64(0), result.CurrentStep)
				assert.Len(t, result.StepExecutionMetadata, 1)
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, result.StepExecutionMetadata[0].Status)
				assert.NotNil(t, result.StepExecutionMetadata[0].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[0].FinishedAt)
			},
		},
		{
			name:  "step is still running; timeout not elapsed",
			steps: []Step{{Kind: "running-step"}},
			assertions: func(t *testing.T, result Result, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseRunning, result.Status)
				assert.Equal(t, int64(0), result.CurrentStep)
				assert.Len(t, result.StepExecutionMetadata, 1)
				assert.Equal(t, kargoapi.PromotionStepStatusRunning, result.StepExecutionMetadata[0].Status)
				assert.NotNil(t, result.StepExecutionMetadata[0].StartedAt)
				assert.Nil(t, result.StepExecutionMetadata[0].FinishedAt)
			},
		},
		{
			name:  "context cancellation",
			steps: []Step{{Kind: "context-waiter"}},
			assertions: func(t *testing.T, result Result, err error) {
				assert.ErrorContains(t, err, "met error threshold")
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Equal(t, int64(0), result.CurrentStep)
				assert.Len(t, result.StepExecutionMetadata, 1)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.StepExecutionMetadata[0].Status)
				assert.Contains(t, result.StepExecutionMetadata[0].Message, context.Canceled.Error())
			},
		},
		{
			name: "output composition step from task",
			stepRunners: []promotion.StepRunner{
				&mockStepRunner{
					name: ComposeOutputStepKind,
					runResult: promotion.StepResult{
						Status: kargoapi.PromotionStepStatusSucceeded,
						Output: map[string]any{"test": "value"},
					},
				},
			},
			steps: []Step{{
				Kind:  ComposeOutputStepKind,
				Alias: "task-1::custom-output",
			}},
			assertions: func(t *testing.T, result Result, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)
				assert.Equal(t, int64(0), result.CurrentStep)
				assert.Len(t, result.StepExecutionMetadata, 1)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.StepExecutionMetadata[0].Status)
				assert.NotNil(t, result.StepExecutionMetadata[0].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[0].FinishedAt)
				assert.Equal(t, promotion.State{
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
			stepRunners: []promotion.StepRunner{
				&mockStepRunner{
					name: ComposeOutputStepKind,
					runResult: promotion.StepResult{
						Status: kargoapi.PromotionStepStatusSucceeded,
						Output: map[string]any{"test": "value"},
					},
				},
			},
			steps: []Step{{
				Kind:  ComposeOutputStepKind,
				Alias: "custom-output",
			}},
			assertions: func(t *testing.T, result Result, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)
				assert.Equal(t, int64(0), result.CurrentStep)
				assert.Len(t, result.StepExecutionMetadata, 1)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.StepExecutionMetadata[0].Status)
				assert.NotNil(t, result.StepExecutionMetadata[0].StartedAt)
				assert.NotNil(t, result.StepExecutionMetadata[0].FinishedAt)
				assert.Equal(t, promotion.State{
					"custom-output": map[string]any{
						"test": "value",
					},
				}, result.State)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			testRegistry := stepRunnerRegistry{}

			var defaultStepRunners = []promotion.StepRunner{
				&mockStepRunner{
					name: "success-step",
					runResult: promotion.StepResult{
						Status: kargoapi.PromotionStepStatusSucceeded,
						Output: map[string]any{"key": "value"},
					},
				},
				&mockStepRunner{
					name: "running-step",
					runResult: promotion.StepResult{
						Status: kargoapi.PromotionStepStatusRunning,
					},
				},
				&mockStepRunner{
					name:      "error-step",
					runResult: promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
					runErr:    errors.New("something went wrong"),
				},
				&mockStepRunner{
					name:      "terminal-error-step",
					runResult: promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
					runErr:    &promotion.TerminalError{Err: errors.New("something went wrong")},
				},
				&mockStepRunner{
					name: "context-waiter",
					runFunc: func(ctx context.Context, _ *promotion.StepContext) (promotion.StepResult, error) {
						cancel()
						<-ctx.Done()
						return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, ctx.Err()
					},
				},
			}
			for _, r := range append(defaultStepRunners, tt.stepRunners...) {
				testRegistry.register(r)
			}

			engine := &simpleEngine{
				registry:    testRegistry,
				kargoClient: fake.NewClientBuilder().Build(),
			}

			result, err := engine.executeSteps(ctx, tt.promoCtx, tt.steps, t.TempDir())
			tt.assertions(t, result, err)
		})
	}
}

func TestSimpleEngine_executeStep(t *testing.T) {
	tests := []struct {
		name       string
		promoCtx   Context
		step       Step
		runner     promotion.StepRunner
		assertions func(*testing.T, promotion.StepResult, error)
	}{
		{
			name: "successful step execution",
			runner: &mockStepRunner{
				name: "success-step",
				runResult: promotion.StepResult{
					Status: kargoapi.PromotionStepStatusSucceeded,
				},
			},
			assertions: func(t *testing.T, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)
			},
		},
		{
			name: "step execution failure",
			step: Step{Kind: "error-step"},
			runner: &mockStepRunner{
				name: "error-step",
				runResult: promotion.StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				},
				runErr: errors.New("something went wrong"),
			},
			assertions: func(t *testing.T, result promotion.StepResult, err error) {
				assert.ErrorContains(t, err, "failed to run step \"error-step\"")
				assert.ErrorContains(t, err, "something went wrong")
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &simpleEngine{
				kargoClient: fake.NewClientBuilder().Build(),
			}

			result, err := engine.executeStep(
				context.Background(),
				tt.promoCtx,
				tt.step,
				tt.runner,
				t.TempDir(),
				make(promotion.State),
			)
			tt.assertions(t, result, err)
		})
	}
}

func TestSimpleEngine_prepareStepContext(t *testing.T) {
	tests := []struct {
		name       string
		promoCtx   Context
		step       Step
		assertions func(*testing.T, *promotion.StepContext, error)
	}{
		{
			name: "successful context preparation",
			promoCtx: Context{
				Project:   "test-project",
				Stage:     "test-stage",
				UIBaseURL: "http://test",
			},
			step: Step{Kind: "test-step"},
			assertions: func(t *testing.T, ctx *promotion.StepContext, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "test-project", ctx.Project)
				assert.Equal(t, "test-stage", ctx.Stage)
				assert.Equal(t, "http://test", ctx.UIBaseURL)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &simpleEngine{
				registry:    stepRunnerRegistry{},
				kargoClient: fake.NewClientBuilder().Build(),
			}

			stepCtx, err := engine.prepareStepContext(
				context.Background(),
				tt.promoCtx,
				tt.step,
				t.TempDir(),
				make(promotion.State),
			)
			tt.assertions(t, stepCtx, err)
		})
	}
}

func TestSimpleEngine_stepAlias(t *testing.T) {
	tests := []struct {
		name       string
		alias      string
		index      int64
		assertions func(*testing.T, string, error)
	}{
		{
			name:  "use provided alias",
			alias: "custom-step",
			assertions: func(t *testing.T, alias string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "custom-step", alias)
			},
		},
		{
			name:  "generate default alias",
			index: 42,
			assertions: func(t *testing.T, alias string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "step-42", alias)
			},
		},
		{
			name:  "reject reserved alias",
			alias: "step-1",
			assertions: func(t *testing.T, _ string, err error) {
				assert.ErrorContains(t, err, "forbidden")
			},
		},
		{
			name:  "trim whitespace",
			alias: "  step  ",
			assertions: func(t *testing.T, alias string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "step", alias)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &simpleEngine{}
			alias, err := engine.stepAlias(tt.alias, tt.index)
			tt.assertions(t, alias, err)
		})
	}
}

func TestSimpleEngine_setupWorkDir(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		existingDir string
		assertions  func(*testing.T, string, error)
	}{
		{
			name:        "use existing directory",
			existingDir: tmpDir,
			assertions: func(t *testing.T, dir string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, tmpDir, dir)
			},
		},
		{
			name: "create new directory",
			assertions: func(t *testing.T, dir string, err error) {
				assert.NoError(t, err)
				assert.DirExists(t, dir)
				t.Cleanup(func() {
					_ = os.RemoveAll(dir)
				})
				assert.Contains(t, dir, "run-")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &simpleEngine{}
			dir, err := engine.setupWorkDir(tt.existingDir)
			tt.assertions(t, dir, err)
		})
	}
}

func TestSimpleEngine_getProjectSecrets(t *testing.T) {
	testData := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
	}
	tests := []struct {
		name        string
		project     string
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, map[string]map[string]string, error)
	}{
		{
			name:    "successful retrieval",
			project: "test-project",
			objects: []client.Object{
				&corev1.Secret{ // Not labeled; should not be included
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret-a",
						Namespace: "test-project",
					},
					Data: testData,
				},
				&corev1.Secret{ // Labeled; should be included
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret-b",
						Namespace: "test-project",
						Labels: map[string]string{
							kargoapi.CredentialTypeLabelKey: kargoapi.CredentialTypeLabelGeneric,
						},
					},
					Data: testData,
				},
			},
			assertions: func(t *testing.T, secrets map[string]map[string]string, err error) {
				assert.NoError(t, err)
				require.Len(t, secrets, 1)
				assert.Equal(t, "value1", secrets["test-secret-b"]["key1"])
				assert.Equal(t, "value2", secrets["test-secret-b"]["key2"])
			},
		},
		{
			name:    "list error",
			project: "test-project",
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return errors.New("list error")
				},
			},
			assertions: func(t *testing.T, _ map[string]map[string]string, err error) {
				assert.ErrorContains(t, err, "error listing Secrets")
			},
		},
		{
			name:    "no secrets",
			project: "empty-project",
			assertions: func(t *testing.T, secrets map[string]map[string]string, err error) {
				assert.NoError(t, err)
				assert.Empty(t, secrets)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &simpleEngine{
				kargoClient: fake.NewClientBuilder().
					WithObjects(tt.objects...).
					WithInterceptorFuncs(tt.interceptor).
					Build(),
			}

			secrets, err := engine.getProjectSecrets(context.Background(), tt.project)
			tt.assertions(t, secrets, err)
		})
	}
}
