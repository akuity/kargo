package v1alpha1

import (
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
	promotionRequestCopy.Status.Summary.Running = 2
	promotionRequestCopy.Status.StartedAt = nil

	require.Equal(t, "fake-target", promotionRequest.Spec.Targets[0].Name)
	require.Equal(t, metav1.ConditionTrue, promotionRequest.Status.Conditions[0].Status)
	require.EqualValues(t, 1, promotionRequest.Status.Summary.Running)
	require.Equal(t, &now, promotionRequest.Status.StartedAt)
}
