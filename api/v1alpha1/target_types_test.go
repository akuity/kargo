package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTarget_GetStatus(t *testing.T) {
	t.Parallel()
	target := &Target{}
	require.Same(t, &target.Status, target.GetStatus())
}

func TestTargetStatus_SetStageStatus(t *testing.T) {
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
			status.SetStageStatus(testCase.stage, testCase.set)
			testCase.assert(t, status)
		})
	}
}

func TestTargetStatus_RemoveStageStatus(t *testing.T) {
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
			status.RemoveStageStatus(testCase.stage)
			testCase.assert(t, status)
		})
	}
}
