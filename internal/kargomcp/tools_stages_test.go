package kargomcp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStageToSummary(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		input  stageJSON
		assert func(*testing.T, stageSummary)
	}{
		{
			name: "minimal stage",
			input: func() stageJSON {
				var s stageJSON
				s.Metadata.Name = "dev"
				s.Status.Health.Status = "Healthy"
				s.Status.FreightSummary = "abc123"
				return s
			}(),
			assert: func(t *testing.T, s stageSummary) {
				require.Equal(t, "dev", s.Name)
				require.Equal(t, "Healthy", s.Health)
				require.Equal(t, "abc123", s.CurrentFreight)
				require.Nil(t, s.LastPromotion)
				require.Empty(t, s.CurrentPromotion)
			},
		},
		{
			name: "stage with last promotion",
			input: func() stageJSON {
				var s stageJSON
				s.Metadata.Name = "prod"
				s.Status.Health.Status = "Unknown"
				s.Status.Health.Issues = []string{"reason"}
				s.Status.LastPromotion.Name = "promo-xyz"
				s.Status.LastPromotion.FinishedAt = "2026-01-01T00:00:00Z"
				s.Status.LastPromotion.Status.Phase = "Errored"
				s.Status.LastPromotion.Status.Message = "merge conflict"
				return s
			}(),
			assert: func(t *testing.T, s stageSummary) {
				require.Equal(t, "prod", s.Name)
				require.Equal(t, []string{"reason"}, s.HealthIssues)
				require.NotNil(t, s.LastPromotion)
				require.Equal(t, "promo-xyz", s.LastPromotion.Name)
				require.Equal(t, "Errored", s.LastPromotion.Phase)
				require.Equal(t, "merge conflict", s.LastPromotion.Message)
			},
		},
		{
			name: "stage with active promotion",
			input: func() stageJSON {
				var s stageJSON
				s.Metadata.Name = "staging"
				s.Status.CurrentPromotion.Name = "promo-active"
				return s
			}(),
			assert: func(t *testing.T, s stageSummary) {
				require.Equal(t, "promo-active", s.CurrentPromotion)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.assert(t, stageToSummary(tc.input))
		})
	}
}
