package builtin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion/gate/types"
)

func TestSoakTimeGate(t *testing.T) {
	t.Parallel()

	soakGate := NewSoakTimeGate()
	require.Equal(t, SoakTimeGateName, soakGate.Name())
}

func TestSoakTimeGate_Evaluate(t *testing.T) {
	t.Parallel()

	const (
		targetStage = "target"
		freightName = "freight"
	)
	now := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	requiredSoakTime := &metav1.Duration{Duration: time.Hour}
	denyDecision := &types.Decision{
		Message: `Freight "freight" has not met the 1h0m0s soak requirement for Stage "target"`,
	}
	// newInput assembles a PromotionInput where the Stage requests Freight from
	// the Freight's origin with the given request. The soak gate resolves the
	// applicable request via PromotionInput.FreightRequest, so the origins must
	// line up.
	newInput := func(
		stageName string,
		freight *kargoapi.Freight,
		request kargoapi.FreightRequest,
	) types.PromotionInput {
		freight.Origin = testOrigin
		request.Origin = testOrigin
		return types.PromotionInput{
			Stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{Name: stageName},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{request},
				},
			},
			Freight: freight,
		}
	}
	testCases := []struct {
		name             string
		input            types.PromotionInput
		expectedDecision *types.Decision
		expectedError    string
	}{
		{
			name: "nil Stage",
			input: types.PromotionInput{
				Freight: &kargoapi.Freight{},
			},
			expectedError: "stage is nil",
		},
		{
			name: "nil Freight",
			input: types.PromotionInput{
				Stage: &kargoapi.Stage{},
			},
			expectedError: "freight is nil",
		},
		{
			name: "no soak required",
			input: types.PromotionInput{
				Stage:   &kargoapi.Stage{},
				Freight: &kargoapi.Freight{},
			},
			expectedDecision: types.NewAllowDecision(),
		},
		{
			name: "approval bypasses soak",
			input: newInput(
				targetStage,
				&kargoapi.Freight{
					Status: kargoapi.FreightStatus{
						ApprovedFor: map[string]kargoapi.ApprovedStage{
							targetStage: {},
						},
					},
				},
				freightRequest([]string{"upstream"}, requiredSoakTime, ""),
			),
			expectedDecision: types.NewAllowDecision(),
		},
		{
			name: "direct source bypasses soak",
			input: newInput(
				"",
				&kargoapi.Freight{},
				kargoapi.FreightRequest{
					Sources: kargoapi.FreightSources{
						Direct:           true,
						RequiredSoakTime: requiredSoakTime,
					},
				},
			),
			expectedDecision: types.NewAllowDecision(),
		},
		{
			name: "required soak has no upstream Stages",
			input: newInput(
				"",
				&kargoapi.Freight{},
				freightRequest(nil, requiredSoakTime, ""),
			),
			expectedError: "FreightRequest has a soak requirement but no upstream Stages",
		},
		{
			name: "one-of allows completed soak",
			input: newInput(
				"",
				freightWithSoaks(
					"",
					now,
					map[string]soakFixture{
						"upstream": {
							verified:  true,
							completed: duration(time.Hour),
						},
					},
				),
				freightRequest([]string{"upstream"}, requiredSoakTime, ""),
			),
			expectedDecision: types.NewAllowDecision(),
		},
		{
			name: "one-of uses shortest active timer",
			input: newInput(
				targetStage,
				freightWithSoaks(
					freightName,
					now,
					map[string]soakFixture{
						"upstream-a": {
							verified: true,
							current:  duration(20 * time.Minute),
						},
						"upstream-b": {
							verified: true,
							current:  duration(50 * time.Minute),
						},
					},
				),
				freightRequest([]string{"upstream-a", "upstream-b"}, requiredSoakTime, ""),
			),
			expectedDecision: decisionWithRequeue(
				denyDecision,
				10*time.Minute,
			),
		},
		{
			name: "one-of active timer is based on current soak",
			input: newInput(
				targetStage,
				freightWithSoaks(
					freightName,
					now,
					map[string]soakFixture{
						"upstream": {
							verified:  true,
							completed: duration(50 * time.Minute),
							current:   duration(10 * time.Minute),
						},
					},
				),
				freightRequest([]string{"upstream"}, requiredSoakTime, ""),
			),
			expectedDecision: decisionWithRequeue(
				denyDecision,
				50*time.Minute,
			),
		},
		{
			name: "one-of does not requeue without an active verified soak",
			input: newInput(
				targetStage,
				freightWithSoaks(
					freightName,
					now,
					map[string]soakFixture{"upstream": {}},
				),
				freightRequest([]string{"upstream"}, requiredSoakTime, ""),
			),
			expectedDecision: denyDecision,
		},
		{
			// Regression test for
			// https://github.com/akuity/kargo/issues/4586: with an active
			// (CurrentlyIn) soak timer that has already met the requirement,
			// auto-promotion must proceed instead of stalling.
			name: "one-of allows an active timer that has met the soak requirement (issue #4586)",
			input: newInput(
				targetStage,
				freightWithSoaks(
					freightName,
					now,
					map[string]soakFixture{
						"upstream-a": {
							verified: true,
							current:  duration(90 * time.Minute),
						},
						"upstream-b": {
							verified: true,
							current:  duration(20 * time.Minute),
						},
					},
				),
				freightRequest(
					[]string{"upstream-a", "upstream-b"},
					requiredSoakTime,
					kargoapi.FreightAvailabilityStrategyOneOf,
				),
			),
			expectedDecision: types.NewAllowDecision(),
		},
		{
			name: "all requires every upstream soak",
			input: newInput(
				targetStage,
				freightWithSoaks(
					freightName,
					now,
					map[string]soakFixture{
						"upstream-a": {
							verified: true,
							current:  duration(20 * time.Minute),
						},
						"upstream-b": {
							verified: true,
							current:  duration(50 * time.Minute),
						},
					},
				),
				freightRequest(
					[]string{"upstream-a", "upstream-b"},
					requiredSoakTime,
					kargoapi.FreightAvailabilityStrategyAll,
				),
			),
			expectedDecision: decisionWithRequeue(
				denyDecision,
				40*time.Minute,
			),
		},
		{
			name: "all does not requeue when an unmet timer is inactive",
			input: newInput(
				targetStage,
				freightWithSoaks(
					freightName,
					now,
					map[string]soakFixture{
						"upstream-a": {
							verified:  true,
							completed: duration(time.Hour),
						},
						"upstream-b": {
							verified: true,
						},
					},
				),
				freightRequest(
					[]string{"upstream-a", "upstream-b"},
					requiredSoakTime,
					kargoapi.FreightAvailabilityStrategyAll,
				),
			),
			expectedDecision: denyDecision,
		},
		{
			name: "all allows every completed upstream soak",
			input: newInput(
				"",
				freightWithSoaks(
					"",
					now,
					map[string]soakFixture{
						"upstream-a": {
							verified:  true,
							completed: duration(time.Hour),
						},
						"upstream-b": {
							verified:  true,
							completed: duration(time.Hour),
						},
					},
				),
				freightRequest(
					[]string{"upstream-a", "upstream-b"},
					requiredSoakTime,
					kargoapi.FreightAvailabilityStrategyAll,
				),
			),
			expectedDecision: types.NewAllowDecision(),
		},
		{
			// Regression test for
			// https://github.com/akuity/kargo/issues/4586: with All, once
			// every active (CurrentlyIn) soak timer has met the requirement,
			// auto-promotion must proceed instead of being blocked
			// indefinitely.
			name: "all allows active timers that have met the soak requirement (issue #4586)",
			input: newInput(
				"",
				freightWithSoaks(
					"",
					now,
					map[string]soakFixture{
						"upstream-a": {
							verified: true,
							current:  duration(90 * time.Minute),
						},
						"upstream-b": {
							verified: true,
							current:  duration(2 * time.Hour),
						},
					},
				),
				freightRequest(
					[]string{"upstream-a", "upstream-b"},
					requiredSoakTime,
					kargoapi.FreightAvailabilityStrategyAll,
				),
			),
			expectedDecision: types.NewAllowDecision(),
		},
		{
			// Regression test for
			// https://github.com/akuity/kargo/issues/4586: a deny for an
			// in-progress soak must carry a RequeueAfter so the controller
			// re-evaluates when the soak completes, rather than relying on the
			// periodic reconcile interval (which left auto-promotion stuck).
			name: "all requeues an in-progress soak so auto-promotion is not stuck (issue #4586)",
			input: newInput(
				targetStage,
				freightWithSoaks(
					freightName,
					now,
					map[string]soakFixture{
						"upstream-a": {
							verified: true,
							current:  duration(55 * time.Minute),
						},
						"upstream-b": {
							verified: true,
							current:  duration(59 * time.Minute),
						},
					},
				),
				freightRequest(
					[]string{"upstream-a", "upstream-b"},
					requiredSoakTime,
					kargoapi.FreightAvailabilityStrategyAll,
				),
			),
			expectedDecision: decisionWithRequeue(
				denyDecision,
				5*time.Minute,
			),
		},
		{
			name: "unsupported availability strategy",
			input: newInput(
				"",
				&kargoapi.Freight{},
				freightRequest([]string{"upstream"}, requiredSoakTime, "unsupported"),
			),
			expectedError: `unsupported Freight availability strategy "unsupported"`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			soakGate := &soakTimeGate{
				nowFn: func() time.Time { return now },
			}
			decision, err := soakGate.Evaluate(t.Context(), testCase.input)
			if testCase.expectedError != "" {
				require.EqualError(t, err, testCase.expectedError)
				require.NotNil(t, decision)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.expectedDecision, decision)
		})
	}
}

type soakFixture struct {
	verified  bool
	completed *time.Duration
	current   *time.Duration
}

func freightWithSoaks(
	name string,
	now time.Time,
	fixtures map[string]soakFixture,
) *kargoapi.Freight {
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: kargoapi.FreightStatus{
			VerifiedIn:  map[string]kargoapi.VerifiedStage{},
			CurrentlyIn: map[string]kargoapi.CurrentStage{},
		},
	}
	for stage, fixture := range fixtures {
		if fixture.verified {
			verifiedStage := kargoapi.VerifiedStage{}
			if fixture.completed != nil {
				verifiedStage.LongestCompletedSoak = &metav1.Duration{
					Duration: *fixture.completed,
				}
			}
			freight.Status.VerifiedIn[stage] = verifiedStage
		}
		if fixture.current != nil {
			freight.Status.CurrentlyIn[stage] = kargoapi.CurrentStage{
				Since: &metav1.Time{Time: now.Add(-*fixture.current)},
			}
		}
	}
	return freight
}

func freightRequest(
	stages []string,
	requiredSoakTime *metav1.Duration,
	strategy kargoapi.FreightAvailabilityStrategy,
) kargoapi.FreightRequest {
	return kargoapi.FreightRequest{
		Sources: kargoapi.FreightSources{
			Stages:               stages,
			RequiredSoakTime:     requiredSoakTime,
			AvailabilityStrategy: strategy,
		},
	}
}

func decisionWithRequeue(
	decision *types.Decision,
	requeueAfter time.Duration,
) *types.Decision {
	d := *decision
	d.RequeueAfter = &requeueAfter
	return &d
}

func duration(value time.Duration) *time.Duration {
	return &value
}
