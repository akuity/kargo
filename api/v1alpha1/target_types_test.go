package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTarget_GetStatus(t *testing.T) {
	t.Parallel()
	target := &Target{}
	require.Same(t, &target.Status, target.GetStatus())
}

func TestTargetStatus_SetStatusForStage(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		status TargetStatus
		stage  string
		set    TargetStageStatus
		assert func(*testing.T, TargetStatus)
	}{
		{
			name:  "nil map is initialized",
			stage: "fleet",
			set:   TargetStageStatus{Health: &Health{Status: HealthStateHealthy}},
			assert: func(t *testing.T, s TargetStatus) {
				require.Len(t, s.Stages, 1)
				require.Equal(t, HealthStateHealthy, s.Stages["fleet"].Health.Status)
			},
		},
		{
			name: "existing entry for the Stage is replaced",
			status: TargetStatus{
				Stages: map[string]TargetStageStatus{
					"fleet": {
						CurrentFreight: &FreightCollection{ID: "old"},
						Health:         &Health{Status: HealthStateUnhealthy},
					},
				},
			},
			stage: "fleet",
			set:   TargetStageStatus{CurrentFreight: &FreightCollection{ID: "new"}},
			assert: func(t *testing.T, s TargetStatus) {
				require.Len(t, s.Stages, 1)
				require.Equal(t, "new", s.Stages["fleet"].CurrentFreight.ID)
				require.Nil(t, s.Stages["fleet"].Health)
			},
		},
		{
			name: "entries for other Stages are untouched",
			status: TargetStatus{
				Stages: map[string]TargetStageStatus{
					"fleet": {CurrentFreight: &FreightCollection{ID: "fleet-id"}},
				},
			},
			stage: "fleet-multi",
			set:   TargetStageStatus{CurrentFreight: &FreightCollection{ID: "multi-id"}},
			assert: func(t *testing.T, s TargetStatus) {
				require.Len(t, s.Stages, 2)
				require.Equal(t, "fleet-id", s.Stages["fleet"].CurrentFreight.ID)
				require.Equal(t, "multi-id", s.Stages["fleet-multi"].CurrentFreight.ID)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			status := testCase.status
			status.SetStatusForStage(testCase.stage, testCase.set)
			testCase.assert(t, status)
		})
	}
}

func TestTargetStatus_RemoveStatusForStage(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		status TargetStatus
		stage  string
		assert func(*testing.T, TargetStatus)
	}{
		{
			name:  "nil map is a no-op",
			stage: "fleet",
			assert: func(t *testing.T, s TargetStatus) {
				require.Nil(t, s.Stages)
			},
		},
		{
			name: "absent Stage is a no-op",
			status: TargetStatus{
				Stages: map[string]TargetStageStatus{"fleet": {}},
			},
			stage: "fleet-multi",
			assert: func(t *testing.T, s TargetStatus) {
				require.Len(t, s.Stages, 1)
				require.Contains(t, s.Stages, "fleet")
			},
		},
		{
			name: "only the named Stage is removed",
			status: TargetStatus{
				Stages: map[string]TargetStageStatus{
					"fleet":       {},
					"fleet-multi": {},
				},
			},
			stage: "fleet",
			assert: func(t *testing.T, s TargetStatus) {
				require.Len(t, s.Stages, 1)
				require.NotContains(t, s.Stages, "fleet")
				require.Contains(t, s.Stages, "fleet-multi")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			status := testCase.status
			status.RemoveStatusForStage(testCase.stage)
			testCase.assert(t, status)
		})
	}
}

func TestTarget_DeepCopy(t *testing.T) {
	t.Parallel()
	target := &Target{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "us-east-1",
			Labels: map[string]string{"region": "us"},
		},
		Spec: TargetSpec{
			TargetSet: "fleet",
			Params:    map[string]apiextensionsv1.JSON{"cluster": {Raw: []byte(`"c1"`)}},
		},
		Status: TargetStatus{
			Stages: map[string]TargetStageStatus{"stage": {}},
		},
	}
	copied := target.DeepCopy()
	require.Equal(t, target, copied)
	require.NotSame(t, target, copied)
	copied.Labels["region"] = "eu"
	copied.Spec.Params["cluster"] = apiextensionsv1.JSON{Raw: []byte(`"c2"`)}
	require.Equal(t, "us", target.Labels["region"])
	require.JSONEq(t, `"c1"`, string(target.Spec.Params["cluster"].Raw))
}
