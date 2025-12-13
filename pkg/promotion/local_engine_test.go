package promotion

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestSimpleEngine_Promote(t *testing.T) {
	tests := []struct {
		name       string
		promoCtx   Context
		steps      []Step
		assertions func(*testing.T, Result, error)
	}{
		{
			name: "successful promotion",
			promoCtx: Context{
				Project: "test-project",
				State:   State{"existing": "state"},
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
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
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
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseErrored, result.Status)
				assert.Contains(t, result.Message, context.Canceled.Error())
				assert.Len(t, result.StepExecutionMetadata, 1)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.StepExecutionMetadata[0].Status)
				assert.Contains(t, result.StepExecutionMetadata[0].Message, context.Canceled.Error())
			},
		},
		{
			name: "running promotion with recoverable error returns error for backoff",
			promoCtx: Context{
				Project: "test-project",
			},
			steps: []Step{
				{
					Kind:  "recoverable-error-step",
					Alias: "step1",
					Retry: &kargoapi.PromotionStepRetry{ErrorThreshold: 3},
				},
			},
			assertions: func(t *testing.T, result Result, err error) {
				assert.Error(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseRunning, result.Status)
				assert.Len(t, result.StepExecutionMetadata, 1)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.StepExecutionMetadata[0].Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			testRegistry := MustNewStepRunnerRegistry(
				StepRunnerRegistration{
					Name: "success-step",
					Value: func(StepRunnerCapabilities) StepRunner {
						return &MockStepRunner{
							RunResult: StepResult{
								Status: kargoapi.PromotionStepStatusSucceeded,
							},
						}
					},
				},
				StepRunnerRegistration{
					Name: "error-step",
					Value: func(StepRunnerCapabilities) StepRunner {
						return &MockStepRunner{
							RunResult: StepResult{
								Status: kargoapi.PromotionStepStatusErrored,
							},
							RunErr: errors.New("something went wrong"),
						}
					},
				},
				StepRunnerRegistration{
					Name: "context-waiter",
					Value: func(StepRunnerCapabilities) StepRunner {
						return &MockStepRunner{
							RunFunc: func(
								ctx context.Context,
								_ *StepContext,
							) (StepResult, error) {
								cancel() // Cancel context immediately
								<-ctx.Done()
								return StepResult{
									Status: kargoapi.PromotionStepStatusErrored,
								}, ctx.Err()
							},
						}
					},
				},
				StepRunnerRegistration{
					Name: "recoverable-error-step",
					Value: func(StepRunnerCapabilities) StepRunner {
						return &MockStepRunner{
							RunResult: StepResult{
								Status: kargoapi.PromotionStepStatusErrored,
							},
							RunErr: errors.New("recoverable error"),
						}
					},
				},
			)

			engine := &LocalEngine{
				orchestator: NewLocalOrchestrator(
					testRegistry,
					fake.NewClientBuilder().Build(),
					fake.NewClientBuilder().Build(),
					nil,
					nil,
				),
			}

			result, err := engine.Promote(ctx, tt.promoCtx, tt.steps)
			tt.assertions(t, result, err)
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
			engine := &LocalEngine{}
			dir, err := engine.setupWorkDir(tt.existingDir)
			tt.assertions(t, dir, err)
		})
	}
}
