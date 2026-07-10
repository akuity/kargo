package builtin

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion/gate/types"
)

const (
	testNamespace = "test-project"
	testStageName = "test-stage"
)

var testOrigin = kargoapi.FreightOrigin{
	Kind: kargoapi.FreightOriginKindWarehouse,
	Name: "test-warehouse",
}

func TestNamespaceGate_Evaluate(t *testing.T) {
	t.Parallel()

	g := NewNamespaceGate()
	require.Equal(t, NamespaceGateName, g.Name())

	testCases := []struct {
		name             string
		input            types.PromotionInput
		expectedDecision *types.Decision
		expectedError    string
	}{
		{
			name:          "nil Stage",
			input:         types.PromotionInput{Freight: freight(testNamespace, testOrigin, nil)},
			expectedError: "stage is nil",
		},
		{
			name:          "nil Freight",
			input:         types.PromotionInput{Stage: stage(oneOfRequest())},
			expectedError: "freight is nil",
		},
		{
			name: "different namespaces are denied",
			input: types.PromotionInput{
				Stage:   stage(oneOfRequest()),
				Freight: freight("other-project", testOrigin, nil),
			},
			expectedDecision: types.NewDenyDecision().WithMessage(
				`Freight "test-freight" is in namespace "other-project", ` +
					`but Stage "test-stage" is in namespace "test-project"`,
			),
		},
		{
			name: "matching namespace is allowed",
			input: types.PromotionInput{
				Stage:   stage(oneOfRequest()),
				Freight: freight(testNamespace, testOrigin, nil),
			},
			expectedDecision: types.NewAllowDecision(),
		},
	}
	runGateCases(t, g, testCases)
}

func TestRequestedOriginGate_Evaluate(t *testing.T) {
	t.Parallel()

	g := NewRequestedOriginGate()
	require.Equal(t, RequestedOriginGateName, g.Name())

	testCases := []struct {
		name             string
		input            types.PromotionInput
		expectedDecision *types.Decision
		expectedError    string
	}{
		{
			name:          "nil Stage",
			input:         types.PromotionInput{Freight: freight(testNamespace, testOrigin, nil)},
			expectedError: "stage is nil",
		},
		{
			name:          "nil Freight",
			input:         types.PromotionInput{Stage: stage(oneOfRequest())},
			expectedError: "freight is nil",
		},
		{
			name: "unrequested origin is denied",
			input: types.PromotionInput{
				Stage: stage(kargoapi.FreightRequest{
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "other-warehouse",
					},
					Sources: kargoapi.FreightSources{Direct: true},
				}),
				Freight: freight(testNamespace, testOrigin, nil),
			},
			expectedDecision: types.NewDenyDecision().WithMessage(
				`Stage "test-stage" does not request Freight from ` +
					`Warehouse "test-warehouse"`,
			),
		},
		{
			name: "requested origin is allowed",
			input: types.PromotionInput{
				Stage:   stage(oneOfRequest()),
				Freight: freight(testNamespace, testOrigin, nil),
			},
			expectedDecision: types.NewAllowDecision(),
		},
	}
	runGateCases(t, g, testCases)
}

func TestAvailabilityGate_Evaluate(t *testing.T) {
	t.Parallel()

	g := NewAvailabilityGate()
	require.Equal(t, AvailabilityGateName, g.Name())

	approvedStatus := &kargoapi.FreightStatus{
		ApprovedFor: map[string]kargoapi.ApprovedStage{testStageName: {}},
	}

	testCases := []struct {
		name             string
		input            types.PromotionInput
		expectedDecision *types.Decision
		expectedError    string
	}{
		{
			name:          "nil Stage",
			input:         types.PromotionInput{Freight: freight(testNamespace, testOrigin, nil)},
			expectedError: "stage is nil",
		},
		{
			name:          "nil Freight",
			input:         types.PromotionInput{Stage: stage(oneOfRequest())},
			expectedError: "freight is nil",
		},
		{
			name: "unrequested origin is denied even when approved",
			input: types.PromotionInput{
				Stage: stage(kargoapi.FreightRequest{
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "other-warehouse",
					},
					Sources: kargoapi.FreightSources{Stages: []string{"upstream"}},
				}),
				Freight: freight(testNamespace, testOrigin, approvedStatus),
			},
			expectedDecision: types.NewDenyDecision().WithMessage(
				`Freight "test-freight" is not available to Stage "test-stage"`,
			),
		},
		{
			name: "approval satisfies availability",
			input: types.PromotionInput{
				Stage:   stage(oneOfRequest()),
				Freight: freight(testNamespace, testOrigin, approvedStatus),
			},
			expectedDecision: types.NewAllowDecision(),
		},
		{
			name: "direct source satisfies availability",
			input: types.PromotionInput{
				Stage: stage(kargoapi.FreightRequest{
					Origin:  testOrigin,
					Sources: kargoapi.FreightSources{Direct: true},
				}),
				Freight: freight(testNamespace, testOrigin, nil),
			},
			expectedDecision: types.NewAllowDecision(),
		},
		{
			name: "one-of requires upstream verification",
			input: types.PromotionInput{
				Stage:   stage(oneOfRequest()),
				Freight: freight(testNamespace, testOrigin, nil),
			},
			expectedDecision: types.NewDenyDecision().WithMessage(
				`Freight "test-freight" must be verified in at least one ` +
					`upstream Stage (upstream) to be available to Stage "test-stage"`,
			),
		},
		{
			name: "one-of upstream verification satisfies availability",
			input: types.PromotionInput{
				Stage: stage(oneOfRequest()),
				Freight: freight(testNamespace, testOrigin, &kargoapi.FreightStatus{
					VerifiedIn: map[string]kargoapi.VerifiedStage{"upstream": {}},
				}),
			},
			expectedDecision: types.NewAllowDecision(),
		},
		{
			name: "all requires every upstream verification",
			input: types.PromotionInput{
				Stage: stage(allRequest("upstream-a", "upstream-b")),
				Freight: freight(testNamespace, testOrigin, &kargoapi.FreightStatus{
					VerifiedIn: map[string]kargoapi.VerifiedStage{"upstream-a": {}},
				}),
			},
			expectedDecision: types.NewDenyDecision().WithMessage(
				`Freight "test-freight" must be verified in all upstream ` +
					`Stages (upstream-a, upstream-b) to be available to ` +
					`Stage "test-stage"`,
			),
		},
		{
			name: "all upstream verifications satisfy availability",
			input: types.PromotionInput{
				Stage: stage(allRequest("upstream-a", "upstream-b")),
				Freight: freight(testNamespace, testOrigin, &kargoapi.FreightStatus{
					VerifiedIn: map[string]kargoapi.VerifiedStage{
						"upstream-a": {},
						"upstream-b": {},
					},
				}),
			},
			expectedDecision: types.NewAllowDecision(),
		},
	}
	runGateCases(t, g, testCases)
}

func runGateCases(
	t *testing.T,
	g types.PromotionGate,
	testCases []struct {
		name             string
		input            types.PromotionInput
		expectedDecision *types.Decision
		expectedError    string
	},
) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			decision, err := g.Evaluate(t.Context(), testCase.input)
			if testCase.expectedError != "" {
				require.EqualError(t, err, testCase.expectedError)
				require.Nil(t, decision)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.expectedDecision, decision)
		})
	}
}

func oneOfRequest() kargoapi.FreightRequest {
	return kargoapi.FreightRequest{
		Origin:  testOrigin,
		Sources: kargoapi.FreightSources{Stages: []string{"upstream"}},
	}
}

func allRequest(stages ...string) kargoapi.FreightRequest {
	return kargoapi.FreightRequest{
		Origin: testOrigin,
		Sources: kargoapi.FreightSources{
			Stages:               stages,
			AvailabilityStrategy: kargoapi.FreightAvailabilityStrategyAll,
		},
	}
}

func stage(request kargoapi.FreightRequest) *kargoapi.Stage {
	return &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testStageName,
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{request},
		},
	}
}

func freight(
	namespace string,
	origin kargoapi.FreightOrigin,
	status *kargoapi.FreightStatus,
) *kargoapi.Freight {
	f := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "test-freight",
		},
		Origin: origin,
	}
	if status != nil {
		f.Status = *status
	}
	return f
}
