package promotion

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/component"
	"github.com/akuity/kargo/pkg/credentials"
)

func TestNewLocalStepExecutor(t *testing.T) {
	registry := MustNewStepRunnerRegistry(
		StepRunnerRegistration{
			Name: "fake-step",
			Value: func(_ StepRunnerCapabilities) StepRunner {
				return &MockStepRunner{}
			},
		},
	)

	kargoClient := fake.NewClientBuilder().Build()
	argoCDClient := fake.NewClientBuilder().Build()
	credsDB := &credentials.FakeDB{}

	executor := NewLocalStepExecutor(
		registry,
		kargoClient,
		argoCDClient,
		credsDB,
	)

	require.NotNil(t, executor)
	require.IsType(t, &LocalStepExecutor{}, executor)
	require.Equal(t, registry, executor.registry)
	require.Equal(t, kargoClient, executor.kargoClient)
	require.Equal(t, argoCDClient, executor.argoCDClient)
	require.Equal(t, credsDB, executor.credsDB)
}

func TestLocalStepExecutor_ExecuteStep(t *testing.T) {
	tests := []struct {
		name       string
		registry   StepRunnerRegistry
		request    StepExecutionRequest
		assertions func(t *testing.T, result StepResult, err error)
	}{
		{
			name:     "no runner registered for step kind",
			registry: MustNewStepRunnerRegistry(),
			request: StepExecutionRequest{
				Context: StepContext{},
				Step: Step{
					Kind: "unknown-step",
				},
			},
			assertions: func(t *testing.T, result StepResult, err error) {
				require.Error(t, err)
				require.True(t, component.IsNotFoundError(err))
				require.Equal(t, StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}, result)
			},
		},
		{
			name: "successful step execution",
			registry: MustNewStepRunnerRegistry(
				StepRunnerRegistration{
					Name: "test-step",
					Value: func(_ StepRunnerCapabilities) StepRunner {
						return &MockStepRunner{
							RunResult: StepResult{
								Status: kargoapi.PromotionStepStatusSucceeded,
							},
						}
					},
				},
			),
			request: StepExecutionRequest{
				Context: StepContext{},
				Step: Step{
					Kind: "test-step",
				},
			},
			assertions: func(t *testing.T, result StepResult, err error) {
				require.Equal(t, StepResult{
					Status: kargoapi.PromotionStepStatusSucceeded,
				}, result)
				require.NoError(t, err)
			},
		},
		{
			name: "step execution returns error",
			registry: MustNewStepRunnerRegistry(
				StepRunnerRegistration{
					Name: "test-step",
					Value: func(_ StepRunnerCapabilities) StepRunner {
						return &MockStepRunner{
							RunResult: StepResult{
								Status: kargoapi.PromotionStepStatusErrored,
							},
							RunErr: errors.New("step execution failed"),
						}
					},
				},
			),
			request: StepExecutionRequest{
				Context: StepContext{},
				Step: Step{
					Kind: "test-step",
				},
			},
			assertions: func(t *testing.T, result StepResult, err error) {
				require.Equal(t, StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}, result)
				require.ErrorContains(t, err, "step execution failed")
			},
		},
		{
			name: "step execution panics",
			registry: MustNewStepRunnerRegistry(
				StepRunnerRegistration{
					Name: "test-step",
					Value: func(_ StepRunnerCapabilities) StepRunner {
						return &MockStepRunner{
							RunFunc: func(context.Context, *StepContext) (StepResult, error) {
								panic("step runner panicked")
							},
						}
					},
				},
			),
			request: StepExecutionRequest{
				Context: StepContext{},
				Step: Step{
					Kind: "test-step",
				},
			},
			assertions: func(t *testing.T, result StepResult, err error) {
				require.Equal(t, StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}, result)
				require.ErrorContains(t, err, "step panicked: step runner panicked")

				var terminalErr *TerminalError
				require.ErrorAs(t, err, &terminalErr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewLocalStepExecutor(tt.registry, nil, nil, nil)
			result, err := executor.ExecuteStep(context.Background(), tt.request)
			tt.assertions(t, result, err)
		})
	}
}
