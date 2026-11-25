package promotion

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/telemetry"
)

var (
	spanRecorder     = tracetest.NewSpanRecorder()
	spanRecorderOnce sync.Once
)

// installSpanRecorder makes the global tracer provider record spans in memory
// and returns the recorder, cleared of anything recorded so far.
//
// The provider is installed once per test binary and never swapped out again:
// this package's tracer is package-level, and the OpenTelemetry global tracer
// binds to the first provider it sees, so a provider installed per test would
// only be observed by the first test to run.
func installSpanRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	spanRecorderOnce.Do(func() {
		otel.SetTracerProvider(sdktrace.NewTracerProvider(
			sdktrace.WithSpanProcessor(spanRecorder),
		))
	})
	spanRecorder.Reset()
	return spanRecorder
}

type fakeOrchestrator struct {
	executeStepsFn func(context.Context, Context, []Step) (Result, error)
}

func (f *fakeOrchestrator) ExecuteSteps(
	ctx context.Context,
	promoCtx Context,
	steps []Step,
) (Result, error) {
	return f.executeStepsFn(ctx, promoCtx, steps)
}

func TestLocalEngine_Promote_tracing(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	testCases := []struct {
		name         string
		orchestrator Orchestrator
		assert       func(*testing.T, Result, error, sdktrace.ReadOnlySpan)
	}{
		{
			name: "success",
			orchestrator: &fakeOrchestrator{
				executeStepsFn: func(ctx context.Context, _ Context, _ []Step) (Result, error) {
					// The orchestrator must see the span through its context.
					require.True(t, trace.SpanContextFromContext(ctx).IsValid())
					return Result{Status: kargoapi.PromotionPhaseSucceeded}, nil
				},
			},
			assert: func(t *testing.T, res Result, err error, span sdktrace.ReadOnlySpan) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionPhaseSucceeded, res.Status)
				require.Equal(t, codes.Unset, span.Status().Code)
			},
		},
		{
			name: "error",
			orchestrator: &fakeOrchestrator{
				executeStepsFn: func(context.Context, Context, []Step) (Result, error) {
					return Result{Status: kargoapi.PromotionPhaseErrored},
						errors.New("something went wrong")
				},
			},
			assert: func(t *testing.T, _ Result, err error, span sdktrace.ReadOnlySpan) {
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, codes.Error, span.Status().Code)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := installSpanRecorder(t)
			engine := &LocalEngine{orchestator: testCase.orchestrator}
			res, err := engine.Promote(
				context.Background(),
				Context{
					WorkDir:   t.TempDir(),
					Project:   "fake-project",
					Stage:     "fake-stage",
					Promotion: "fake-promotion",
				},
				nil,
			)
			ended := recorder.Ended()
			require.Len(t, ended, 1)
			span := ended[0]
			require.Equal(t, "Promote", span.Name())
			require.Subset(
				t,
				span.Attributes(),
				[]any{
					telemetry.ProjectKey.String("fake-project"),
					telemetry.StageKey.String("fake-stage"),
					telemetry.PromotionKey.String("fake-promotion"),
				},
			)
			testCase.assert(t, res, err, span)
		})
	}
}

func TestLocalStepExecutor_ExecuteStep_tracing(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	testCases := []struct {
		name   string
		runner StepRunner
		assert func(*testing.T, StepResult, error, sdktrace.ReadOnlySpan)
	}{
		{
			name: "success",
			runner: &MockStepRunner{
				RunFunc: func(ctx context.Context, _ *StepContext) (StepResult, error) {
					// The runner must see the span through its context.
					require.True(t, trace.SpanContextFromContext(ctx).IsValid())
					return StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
				},
			},
			assert: func(t *testing.T, res StepResult, err error, span sdktrace.ReadOnlySpan) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.Contains(
					t,
					span.Attributes(),
					telemetry.StepStatusKey.String(string(kargoapi.PromotionStepStatusSucceeded)),
				)
				require.Equal(t, codes.Unset, span.Status().Code)
			},
		},
		{
			name: "failure",
			runner: &MockStepRunner{
				RunFunc: func(context.Context, *StepContext) (StepResult, error) {
					return StepResult{Status: kargoapi.PromotionStepStatusFailed},
						errors.New("something went wrong")
				},
			},
			assert: func(t *testing.T, res StepResult, err error, span sdktrace.ReadOnlySpan) {
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
				require.Contains(
					t,
					span.Attributes(),
					telemetry.StepStatusKey.String(string(kargoapi.PromotionStepStatusFailed)),
				)
				require.Equal(t, codes.Error, span.Status().Code)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := installSpanRecorder(t)
			executor := NewLocalStepExecutor(
				MustNewStepRunnerRegistry(StepRunnerRegistration{
					Name: "fake-step",
					Value: func(StepRunnerCapabilities) StepRunner {
						return testCase.runner
					},
				}),
				nil,
				nil,
				nil,
				nil,
			)
			res, err := executor.ExecuteStep(
				context.Background(),
				StepExecutionRequest{
					Context: StepContext{Project: "fake-project"},
					Step:    Step{Kind: "fake-step", Alias: "fake-alias"},
				},
			)
			ended := recorder.Ended()
			require.Len(t, ended, 1)
			span := ended[0]
			require.Equal(t, "Execute step", span.Name())
			require.Subset(
				t,
				span.Attributes(),
				[]any{
					telemetry.ProjectKey.String("fake-project"),
					telemetry.StepKindKey.String("fake-step"),
					telemetry.StepAliasKey.String("fake-alias"),
				},
			)
			testCase.assert(t, res, err, span)
		})
	}
}
