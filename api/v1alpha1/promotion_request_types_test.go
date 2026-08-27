package v1alpha1

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPromotionRequest_GetStatus(t *testing.T) {
	t.Parallel()
	promotionRequest := &PromotionRequest{
		Status: PromotionRequestStatus{Phase: PromotionRequestPhaseRunning},
	}
	status := promotionRequest.GetStatus()
	require.Same(t, &promotionRequest.Status, status)
	require.Equal(t, PromotionRequestPhaseRunning, status.Phase)
}

func TestPromotionRequestStatus_GetConditions(t *testing.T) {
	t.Parallel()
	conditions := []metav1.Condition{{
		Type:   ConditionTypeReady,
		Status: metav1.ConditionTrue,
	}}
	status := &PromotionRequestStatus{Conditions: conditions}
	require.Equal(t, conditions, status.GetConditions())
}

func TestPromotionRequestStatus_SetConditions(t *testing.T) {
	t.Parallel()
	status := &PromotionRequestStatus{
		Conditions: []metav1.Condition{{
			Type:   ConditionTypeReady,
			Status: metav1.ConditionFalse,
		}},
	}
	conditions := []metav1.Condition{{
		Type:   ConditionTypeReady,
		Status: metav1.ConditionTrue,
	}}
	status.SetConditions(conditions)
	require.Equal(t, conditions, status.Conditions)
}

func TestPromotionRequestPhase_IsTerminal(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		phase    PromotionRequestPhase
		terminal bool
	}{
		{
			name:  "pending",
			phase: PromotionRequestPhasePending,
		},
		{
			name:  "running",
			phase: PromotionRequestPhaseRunning,
		},
		{
			name:     "succeeded",
			phase:    PromotionRequestPhaseSucceeded,
			terminal: true,
		},
		{
			name:     "failed",
			phase:    PromotionRequestPhaseFailed,
			terminal: true,
		},
		{
			name:     "errored",
			phase:    PromotionRequestPhaseErrored,
			terminal: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.terminal, testCase.phase.IsTerminal())
		})
	}
}

// TestPromotionRequestSpec_Immutability guards the spec's mutability contract:
// spec.targets may change over the PromotionRequest's lifetime so that Targets
// discovered after creation can be added to an in-flight request, while the
// fields identifying the freight transition are fixed once set.
//
// The contract is enforced at admission by CEL transition rules in the CRD.
// This asserts on the markers that generate them, in the same style as
// TestKubernetesNamePattern, so that adding or dropping one is a test failure
// rather than a silent behavior change.
func TestPromotionRequestSpec_Immutability(t *testing.T) {
	t.Parallel()

	const immutabilityMarker = `+kubebuilder:validation:XValidation:rule="self == oldSelf"`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "promotion_request_types.go", nil, parser.ParseComments)
	require.NoError(t, err)

	immutable := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok || typeSpec.Name.Name != "PromotionRequestSpec" {
			return true
		}
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return false
		}
		for _, field := range structType.Fields.List {
			if len(field.Names) == 0 || field.Doc == nil {
				continue
			}
			immutable[field.Names[0].Name] = strings.Contains(field.Doc.Text(), immutabilityMarker)
		}
		return false
	})

	require.Equal(
		t,
		map[string]bool{
			"Stage":   true,
			"Freight": true,
			// The one mutable field. The governing Stage owns it and may add
			// Targets to a PromotionRequest that is still in flight.
			"Targets": false,
		},
		immutable,
	)
}

func TestPromotionRequest_DeepCopy(t *testing.T) {
	t.Parallel()

	now := metav1.Now()
	promotionRequest := &PromotionRequest{
		Spec: PromotionRequestSpec{
			Stage:   "fake-stage",
			Freight: "fake-freight",
			Targets: []PromotionRequestTarget{{Name: "fake-target"}},
		},
		Status: PromotionRequestStatus{
			Conditions: []metav1.Condition{{
				Type:   ConditionTypeReady,
				Status: metav1.ConditionTrue,
			}},
			Phase: PromotionRequestPhaseRunning,
			Targets: []PromotionRequestTargetStatus{{
				Name:      "fake-target",
				Promotion: "fake-stage.fake-target.fake-promotion",
				Phase:     PromotionPhaseRunning,
			}},
			Summary: &PromotionRequestSummary{
				Running: 1,
			},
			StartedAt: &now,
		},
	}

	promotionRequestCopy := promotionRequest.DeepCopy()
	require.Equal(t, promotionRequest, promotionRequestCopy)

	promotionRequestCopy.Spec.Targets[0].Name = "different-target"
	promotionRequestCopy.Status.Conditions[0].Status = metav1.ConditionFalse
	promotionRequestCopy.Status.Targets[0].Phase = PromotionPhaseSucceeded
	promotionRequestCopy.Status.Summary.Running = 2
	promotionRequestCopy.Status.StartedAt = nil

	require.Equal(t, "fake-target", promotionRequest.Spec.Targets[0].Name)
	require.Equal(t, metav1.ConditionTrue, promotionRequest.Status.Conditions[0].Status)
	require.Equal(t, PromotionPhaseRunning, promotionRequest.Status.Targets[0].Phase)
	require.EqualValues(t, 1, promotionRequest.Status.Summary.Running)
	require.Equal(t, &now, promotionRequest.Status.StartedAt)
}
