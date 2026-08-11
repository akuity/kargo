package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPromotionSet_GetStatus(t *testing.T) {
	t.Parallel()
	promotionSet := &PromotionSet{
		Status: PromotionSetStatus{TargetCount: 1},
	}
	status := promotionSet.GetStatus()
	require.Same(t, &promotionSet.Status, status)
	require.EqualValues(t, 1, status.TargetCount)
}

func TestPromotionSet_GetConditions(t *testing.T) {
	t.Parallel()
	conditions := []metav1.Condition{{
		Type:   ConditionTypeReady,
		Status: metav1.ConditionTrue,
	}}
	promotionSet := &PromotionSet{
		Status: PromotionSetStatus{Conditions: conditions},
	}
	require.Equal(t, conditions, promotionSet.GetConditions())
}

func TestPromotionSet_SetConditions(t *testing.T) {
	t.Parallel()
	promotionSet := &PromotionSet{
		Status: PromotionSetStatus{
			Conditions: []metav1.Condition{{
				Type:   ConditionTypeReady,
				Status: metav1.ConditionFalse,
			}},
		},
	}
	conditions := []metav1.Condition{{
		Type:   ConditionTypeReady,
		Status: metav1.ConditionTrue,
	}}
	promotionSet.SetConditions(conditions)
	require.Equal(t, conditions, promotionSet.Status.Conditions)
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
