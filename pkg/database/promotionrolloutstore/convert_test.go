package promotionrolloutstore

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

func TestTargetStatuses(t *testing.T) {
	t.Parallel()
	row := func(target, promotion, phase string) database.ListPromotionRolloutTargetStatusesRow {
		return database.ListPromotionRolloutTargetStatusesRow{
			TargetName:    target,
			PromotionName: promotion,
			Phase:         phase,
		}
	}
	testCases := []struct {
		name   string
		rows   []database.ListPromotionRolloutTargetStatusesRow
		assert func(*testing.T, []kargoapi.PromotionRolloutTargetStatus, *kargoapi.PromotionRolloutSummary)
	}{
		{
			name: "no rows",
			assert: func(
				t *testing.T,
				statuses []kargoapi.PromotionRolloutTargetStatus,
				summary *kargoapi.PromotionRolloutSummary,
			) {
				require.Nil(t, statuses)
				require.Nil(t, summary)
			},
		},
		{
			name: "one entry per row in row order, tallied by phase",
			rows: []database.ListPromotionRolloutTargetStatusesRow{
				row("eu-west", "prod.eu-west.01A.abc", "Pending"),
				row("us-east", "prod.us-east.01C.abc", "Succeeded"),
				row("us-west", "prod.us-west.01A.abc", "Running"),
				row("us-x", "prod.us-x.01A.abc", "Failed"),
				row("us-y", "prod.us-y.01A.abc", "Errored"),
				row("us-z", "prod.us-z.01A.abc", "Aborted"),
			},
			assert: func(
				t *testing.T,
				statuses []kargoapi.PromotionRolloutTargetStatus,
				summary *kargoapi.PromotionRolloutSummary,
			) {
				require.Equal(t, []kargoapi.PromotionRolloutTargetStatus{
					{Name: "eu-west", Promotion: "prod.eu-west.01A.abc", Phase: kargoapi.PromotionPhasePending},
					{Name: "us-east", Promotion: "prod.us-east.01C.abc", Phase: kargoapi.PromotionPhaseSucceeded},
					{Name: "us-west", Promotion: "prod.us-west.01A.abc", Phase: kargoapi.PromotionPhaseRunning},
					{Name: "us-x", Promotion: "prod.us-x.01A.abc", Phase: kargoapi.PromotionPhaseFailed},
					{Name: "us-y", Promotion: "prod.us-y.01A.abc", Phase: kargoapi.PromotionPhaseErrored},
					{Name: "us-z", Promotion: "prod.us-z.01A.abc", Phase: kargoapi.PromotionPhaseAborted},
				}, statuses)
				require.Equal(t, &kargoapi.PromotionRolloutSummary{
					Pending:   1,
					Running:   1,
					Succeeded: 1,
					Failed:    1,
					Errored:   1,
					Aborted:   1,
				}, summary)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			statuses, summary := TargetStatuses(testCase.rows)
			testCase.assert(t, statuses, summary)
		})
	}
}
