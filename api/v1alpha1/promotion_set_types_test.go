package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPromotionSet_GetStatus(t *testing.T) {
	t.Parallel()
	promotionSet := &PromotionSet{
		Status: PromotionSetStatus{Phase: PromotionSetPhaseRunning},
	}
	status := promotionSet.GetStatus()
	require.Same(t, &promotionSet.Status, status)
	require.Equal(t, PromotionSetPhaseRunning, status.Phase)
}

func TestPromotionSetStatus_GetConditions(t *testing.T) {
	t.Parallel()
	conditions := []metav1.Condition{{
		Type:   ConditionTypeReady,
		Status: metav1.ConditionTrue,
	}}
	status := &PromotionSetStatus{Conditions: conditions}
	require.Equal(t, conditions, status.GetConditions())
}

func TestPromotionSetStatus_SetConditions(t *testing.T) {
	t.Parallel()
	status := &PromotionSetStatus{
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

func TestPromotionSetPhase_IsTerminal(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		phase    PromotionSetPhase
		terminal bool
	}{
		{
			name:  "pending",
			phase: PromotionSetPhasePending,
		},
		{
			name:  "running",
			phase: PromotionSetPhaseRunning,
		},
		{
			name:     "succeeded",
			phase:    PromotionSetPhaseSucceeded,
			terminal: true,
		},
		{
			name:     "failed",
			phase:    PromotionSetPhaseFailed,
			terminal: true,
		},
		{
			name:     "errored",
			phase:    PromotionSetPhaseErrored,
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

func TestPromotionSet_DeepCopy(t *testing.T) {
	t.Parallel()

	now := metav1.Now()
	promotionSet := &PromotionSet{
		Spec: PromotionSetSpec{
			Stage:   "fake-stage",
			Freight: "fake-freight",
			Targets: []PromotionSetTarget{{Name: "fake-target"}},
		},
		Status: PromotionSetStatus{
			Conditions: []metav1.Condition{{
				Type:   ConditionTypeReady,
				Status: metav1.ConditionTrue,
			}},
			Phase: PromotionSetPhaseRunning,
			Summary: &PromotionSetSummary{
				Running: 1,
			},
			StartedAt: &now,
		},
	}

	promotionSetCopy := promotionSet.DeepCopy()
	require.Equal(t, promotionSet, promotionSetCopy)

	promotionSetCopy.Spec.Targets[0].Name = "different-target"
	promotionSetCopy.Status.Conditions[0].Status = metav1.ConditionFalse
	promotionSetCopy.Status.Summary.Running = 2
	promotionSetCopy.Status.StartedAt = nil

	require.Equal(t, "fake-target", promotionSet.Spec.Targets[0].Name)
	require.Equal(t, metav1.ConditionTrue, promotionSet.Status.Conditions[0].Status)
	require.EqualValues(t, 1, promotionSet.Status.Summary.Running)
	require.Equal(t, &now, promotionSet.Status.StartedAt)
}
