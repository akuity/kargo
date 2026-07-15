package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTarget_GetStatus(t *testing.T) {
	t.Parallel()
	target := &Target{
		Status: TargetStatus{
			OwnedBy: []TargetOwnership{{Stage: "fake-stage"}},
		},
	}
	status := target.GetStatus()
	require.Same(t, &target.Status, status)
	require.Equal(t, "fake-stage", status.OwnedBy[0].Stage)
}

func TestTargetStatus_GetConditions(t *testing.T) {
	t.Parallel()
	conditions := []metav1.Condition{{
		Type:   ConditionTypeReady,
		Status: metav1.ConditionTrue,
	}}
	status := &TargetStatus{Conditions: conditions}
	require.Equal(t, conditions, status.GetConditions())
}

func TestTargetStatus_SetConditions(t *testing.T) {
	t.Parallel()
	status := &TargetStatus{
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
