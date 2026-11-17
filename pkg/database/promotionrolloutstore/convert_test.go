package promotionrolloutstore

import (
	"encoding/json"
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

func TestFreightCollection(t *testing.T) {
	t.Parallel()
	images := kargoapi.FreightReference{
		Name:   "images-1",
		Origin: kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "images"},
		Images: []kargoapi.Image{{RepoURL: "example.com/app", Tag: "v1.2.3", Digest: "sha256:abc"}},
	}
	config := kargoapi.FreightReference{
		Name:    "config-1",
		Origin:  kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "config"},
		Commits: []kargoapi.GitCommit{{RepoURL: "https://example.com/config.git", ID: "0123abc"}},
		Charts:  []kargoapi.Chart{{RepoURL: "oci://example.com/charts", Name: "app", Version: "4.5.6"}},
	}
	row := func(t *testing.T, ref kargoapi.FreightReference) database.PromotionRolloutFreightRow {
		t.Helper()
		reference, err := json.Marshal(ref)
		require.NoError(t, err)
		return database.PromotionRolloutFreightRow{
			OriginKind:  string(ref.Origin.Kind),
			OriginName:  ref.Origin.Name,
			FreightName: ref.Name,
			Reference:   reference,
		}
	}
	testCases := []struct {
		name   string
		rows   func(*testing.T) []database.PromotionRolloutFreightRow
		assert func(*testing.T, *kargoapi.FreightCollection, error)
	}{
		{
			name: "no rows are an empty collection",
			rows: func(*testing.T) []database.PromotionRolloutFreightRow { return nil },
			assert: func(t *testing.T, collection *kargoapi.FreightCollection, err error) {
				require.NoError(t, err)
				require.Equal(t, &kargoapi.FreightCollection{}, collection)
			},
		},
		{
			name: "each row is its origin's Freight, contents included",
			rows: func(t *testing.T) []database.PromotionRolloutFreightRow {
				return []database.PromotionRolloutFreightRow{row(t, config), row(t, images)}
			},
			assert: func(t *testing.T, collection *kargoapi.FreightCollection, err error) {
				require.NoError(t, err)
				require.Equal(t, map[string]kargoapi.FreightReference{
					"Warehouse/config": config,
					"Warehouse/images": images,
				}, collection.Freight)
				require.Equal(t, []kargoapi.FreightReference{config, images}, collection.References())
				// The collection is identified as one built anywhere else
				// from the same Freight is.
				want := &kargoapi.FreightCollection{}
				want.UpdateOrPush(images, config)
				require.Equal(t, want.ID, collection.ID)
				require.Empty(t, collection.VerificationHistory)
			},
		},
		{
			name: "a reference that is not a FreightReference is an error",
			rows: func(t *testing.T) []database.PromotionRolloutFreightRow {
				bad := row(t, images)
				bad.Reference = json.RawMessage(`{"images": "not a list"}`)
				return []database.PromotionRolloutFreightRow{row(t, config), bad}
			},
			assert: func(t *testing.T, collection *kargoapi.FreightCollection, err error) {
				require.ErrorContains(t, err, `error decoding Freight "images-1" from Warehouse "images"`)
				require.Nil(t, collection)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			collection, err := freightCollection(testCase.rows(t))
			testCase.assert(t, collection, err)
		})
	}
}
