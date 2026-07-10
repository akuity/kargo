package gate

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion/gate/types"
)

func TestNewSet(t *testing.T) {
	t.Parallel()

	require.Equal(t, setName, NewSet().Name())
}

func TestSetEvaluate(t *testing.T) {
	t.Parallel()

	testStage := &kargoapi.Stage{}
	testFreight := &kargoapi.Freight{}
	testInput := types.PromotionInput{
		Stage:   testStage,
		Freight: testFreight,
	}
	denial := types.NewDenyDecision().WithMessage("denied by test gate")
	testCases := []struct {
		name             string
		evaluations      []fakeGateEvaluation
		nilGateAt        int
		expectedDecision *types.Decision
		expectedError    string
		expectedCalls    []string
	}{
		{
			name:             "empty set allows",
			nilGateAt:        -1,
			expectedDecision: types.NewAllowDecision(),
		},
		{
			name: "all gates allow",
			evaluations: []fakeGateEvaluation{
				{decision: types.NewAllowDecision()},
				{decision: types.NewAllowDecision()},
			},
			nilGateAt:        -1,
			expectedDecision: types.NewAllowDecision(),
			expectedCalls:    []string{"gate-0", "gate-1"},
		},
		{
			name: "first denial short-circuits",
			evaluations: []fakeGateEvaluation{
				{decision: types.NewAllowDecision()},
				{decision: denial},
				{err: errors.New("must not be called")},
			},
			nilGateAt:        -1,
			expectedDecision: denial,
			expectedCalls:    []string{"gate-0", "gate-1"},
		},
		{
			name: "first error short-circuits",
			evaluations: []fakeGateEvaluation{
				{decision: types.NewAllowDecision()},
				{err: errors.New("boom")},
				{err: errors.New("must not be called")},
			},
			nilGateAt:     -1,
			expectedError: `error evaluating Promotion creation gate "gate-1": boom`,
			expectedCalls: []string{"gate-0", "gate-1"},
		},
		{
			name: "nil gate is rejected",
			evaluations: []fakeGateEvaluation{
				{decision: types.NewAllowDecision()},
				{decision: types.NewAllowDecision()},
			},
			nilGateAt:     1,
			expectedError: "promotion creation gate at index 1 is nil",
			expectedCalls: []string{"gate-0"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var calls []string
			gates := make(
				[]types.PromotionGate,
				len(testCase.evaluations),
			)
			for i, evaluation := range testCase.evaluations {
				if i == testCase.nilGateAt {
					continue
				}
				gateName := fmt.Sprintf("gate-%d", i)
				gates[i] = &fakeGate{
					name: gateName,
					evaluateFn: func(
						_ context.Context,
						input types.PromotionInput,
					) (*types.Decision, error) {
						require.Same(t, testStage, input.Stage)
						require.Same(t, testFreight, input.Freight)
						calls = append(calls, gateName)
						return evaluation.decision, evaluation.err
					},
				}
			}

			decision, err := NewSet(gates...).Evaluate(
				t.Context(),
				testInput,
			)
			if testCase.expectedError != "" {
				require.EqualError(t, err, testCase.expectedError)
				require.Nil(t, decision)
			} else {
				require.NoError(t, err)
				require.Equal(t, testCase.expectedDecision, decision)
			}
			require.Equal(t, testCase.expectedCalls, calls)
		})
	}
}

type fakeGateEvaluation struct {
	decision *types.Decision
	err      error
}

type fakeGate struct {
	name       string
	evaluateFn func(
		context.Context,
		types.PromotionInput,
	) (*types.Decision, error)
}

func (f *fakeGate) Name() string {
	return f.name
}

func (f *fakeGate) Evaluate(
	ctx context.Context,
	input types.PromotionInput,
) (*types.Decision, error) {
	return f.evaluateFn(ctx, input)
}
