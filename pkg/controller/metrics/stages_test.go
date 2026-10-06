package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_countStagesByReadyReason(t *testing.T) {
	t.Parallel()

	// regular returns a Stage with a promotion template, which is what makes it
	// a regular rather than a control flow Stage.
	regular := func(project string, conditions ...metav1.Condition) kargoapi.Stage {
		return kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Namespace: project},
			Spec: kargoapi.StageSpec{
				PromotionTemplate: &kargoapi.PromotionTemplate{
					Spec: kargoapi.PromotionTemplateSpec{
						Steps: []kargoapi.PromotionStep{{Uses: "fake-step"}},
					},
				},
			},
			Status: kargoapi.StageStatus{Conditions: conditions},
		}
	}

	controlFlow := func(project string, conditions ...metav1.Condition) kargoapi.Stage {
		return kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Namespace: project},
			Status:     kargoapi.StageStatus{Conditions: conditions},
		}
	}

	ready := metav1.Condition{
		Type:   kargoapi.ConditionTypeReady,
		Status: metav1.ConditionTrue,
		Reason: "Verified",
	}
	unhealthy := metav1.Condition{
		Type:   kargoapi.ConditionTypeReady,
		Status: metav1.ConditionFalse,
		Reason: "Unhealthy",
	}

	testCases := []struct {
		name   string
		stages []kargoapi.Stage
		assert func(*testing.T, map[string]map[string]float64)
	}{
		{
			name: "no Stages yields no Projects",
			assert: func(t *testing.T, counts map[string]map[string]float64) {
				require.Empty(t, counts)
			},
		},
		{
			name: "Stages are counted by reason, per Project",
			stages: []kargoapi.Stage{
				regular("project-a", ready),
				regular("project-a", ready),
				regular("project-a", unhealthy),
				regular("project-b", unhealthy),
			},
			assert: func(t *testing.T, counts map[string]map[string]float64) {
				require.Len(t, counts, 2)
				require.Equal(t, float64(2), counts["project-a"]["Verified"])
				require.Equal(t, float64(1), counts["project-a"]["Unhealthy"])
				require.Equal(t, float64(1), counts["project-b"]["Unhealthy"])
			},
		},
		{
			// Unlike the per-condition metric this replaced, a Stage lands in
			// exactly one series, so the series do sum to the Stage count.
			name: "each Stage is counted exactly once",
			stages: []kargoapi.Stage{
				regular("project-a", ready),
				regular("project-a", unhealthy),
			},
			assert: func(t *testing.T, counts map[string]map[string]float64) {
				var total float64
				for _, count := range counts["project-a"] {
					total += count
				}
				require.Equal(t, float64(2), total)
			},
		},
		{
			// A control flow Stage's Ready condition is set directly and never
			// reflects health or verification, so it is left out entirely.
			name: "control flow Stages are omitted",
			stages: []kargoapi.Stage{
				regular("project-a", ready),
				controlFlow("project-a", metav1.Condition{
					Type:   kargoapi.ConditionTypeReady,
					Status: metav1.ConditionTrue,
					Reason: kargoapi.ConditionTypeReady,
				}),
			},
			assert: func(t *testing.T, counts map[string]map[string]float64) {
				require.Len(t, counts["project-a"], 1)
				require.Equal(t, float64(1), counts["project-a"]["Verified"])
			},
		},
		{
			name: "a Project of only control flow Stages is not reported",
			stages: []kargoapi.Stage{
				controlFlow("project-a", ready),
			},
			assert: func(t *testing.T, counts map[string]map[string]float64) {
				require.Empty(t, counts)
			},
		},
		{
			// A Stage the controller has not reconciled yet has no Ready
			// condition at all.
			name:   "a Stage with no Ready condition reports an unknown reason",
			stages: []kargoapi.Stage{regular("project-a")},
			assert: func(t *testing.T, counts map[string]map[string]float64) {
				require.Equal(t, float64(1), counts["project-a"][reasonUnknown])
			},
		},
		{
			// Only Ready is consulted. The conditions it is derived from are
			// ignored, so a Stage never lands in more than one series.
			name: "other conditions are ignored",
			stages: []kargoapi.Stage{
				regular(
					"project-a",
					metav1.Condition{
						Type:   kargoapi.ConditionTypeHealthy,
						Status: metav1.ConditionFalse,
						Reason: "Unhealthy",
					},
					metav1.Condition{
						Type:   kargoapi.ConditionTypePromoting,
						Status: metav1.ConditionTrue,
						Reason: "ActivePromotion",
					},
					metav1.Condition{
						Type:   kargoapi.ConditionTypeReady,
						Status: metav1.ConditionFalse,
						Reason: "ActivePromotion",
					},
				),
			},
			assert: func(t *testing.T, counts map[string]map[string]float64) {
				require.Len(t, counts["project-a"], 1)
				require.Equal(t, float64(1), counts["project-a"]["ActivePromotion"])
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.assert(t, countStagesByReadyReason(testCase.stages))
		})
	}
}
