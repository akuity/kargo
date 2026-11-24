package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPromotionRollout_GetStatus(t *testing.T) {
	t.Parallel()
	rollout := &PromotionRollout{
		Status: PromotionRolloutStatus{Phase: PromotionRolloutPhaseRunning},
	}
	status := rollout.GetStatus()
	require.Same(t, &rollout.Status, status)
	require.Equal(t, PromotionRolloutPhaseRunning, status.Phase)
}

func TestPromotionRolloutStatus_GetConditions(t *testing.T) {
	t.Parallel()
	conditions := []metav1.Condition{{
		Type:   ConditionTypeReady,
		Status: metav1.ConditionTrue,
	}}
	status := &PromotionRolloutStatus{Conditions: conditions}
	require.Equal(t, conditions, status.GetConditions())
}

func TestPromotionRolloutStatus_SetConditions(t *testing.T) {
	t.Parallel()
	status := &PromotionRolloutStatus{
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

func TestPromotionRolloutPhase_IsTerminal(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		phase    PromotionRolloutPhase
		terminal bool
	}{
		{name: "pending", phase: PromotionRolloutPhasePending},
		{name: "running", phase: PromotionRolloutPhaseRunning},
		{name: "succeeded", phase: PromotionRolloutPhaseSucceeded, terminal: true},
		{name: "failed", phase: PromotionRolloutPhaseFailed, terminal: true},
		{name: "errored", phase: PromotionRolloutPhaseErrored, terminal: true},
		{name: "aborted", phase: PromotionRolloutPhaseAborted, terminal: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.terminal, testCase.phase.IsTerminal())
		})
	}
}

func TestPromotionRollout_DeepCopy(t *testing.T) {
	t.Parallel()

	now := metav1.Now()
	rollout := &PromotionRollout{
		Number: 3,
		Spec: PromotionRolloutSpec{
			Stage:   "fake-stage",
			Freight: "fake-freight",
			Targets: []PromotionRolloutTarget{{Name: "fake-target"}},
		},
		Status: PromotionRolloutStatus{
			Conditions: []metav1.Condition{{
				Type:   ConditionTypeReady,
				Status: metav1.ConditionTrue,
			}},
			Phase: PromotionRolloutPhaseRunning,
			Targets: []PromotionRolloutTargetStatus{{
				Name:      "fake-target",
				Promotion: "fake-stage.fake-target.fake-promotion",
				Phase:     PromotionPhaseRunning,
			}},
			Summary:   &PromotionRolloutSummary{Running: 1},
			StartedAt: &now,
		},
	}

	rolloutCopy := rollout.DeepCopy()
	require.Equal(t, rollout, rolloutCopy)

	rolloutCopy.Spec.Targets[0].Name = "different-target"
	rolloutCopy.Status.Conditions[0].Status = metav1.ConditionFalse
	rolloutCopy.Status.Targets[0].Phase = PromotionPhaseSucceeded
	rolloutCopy.Status.Summary.Running = 2
	rolloutCopy.Status.StartedAt = nil

	require.Equal(t, "fake-target", rollout.Spec.Targets[0].Name)
	require.Equal(t, metav1.ConditionTrue, rollout.Status.Conditions[0].Status)
	require.Equal(t, PromotionPhaseRunning, rollout.Status.Targets[0].Phase)
	require.EqualValues(t, 1, rollout.Status.Summary.Running)
	require.Equal(t, &now, rollout.Status.StartedAt)
}
