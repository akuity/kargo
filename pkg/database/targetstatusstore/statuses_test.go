package targetstatusstore

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database"
)

func TestParams(t *testing.T) {
	t.Parallel()
	first := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	second := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	precedence := []string{
		"Promoting", "Pending", "Failed", "NotAttempted", "NeverPromoted", "Behind", "UpToDate",
	}
	testCases := []struct {
		name   string
		scope  Scope
		assert func(*testing.T, database.ListTargetStatusesParams)
	}{
		{
			name: "an empty scope encodes as empty arrays, never NULL ones",
			assert: func(t *testing.T, args database.ListTargetStatusesParams) {
				require.Equal(t, []uuid.UUID{}, args.TargetIds)
				require.Equal(t, []string{}, args.TargetNames)
				require.Equal(t, []string{}, args.StageIds)
				require.Equal(t, []string{}, args.FreightStageIds)
				require.Equal(t, []string{}, args.StageFreight)
				require.Equal(t, []string{}, args.States)
				require.Equal(t, precedence, args.Precedence)
				require.False(t, args.Summarize)
			},
		},
		{
			name: "pairs are zipped into arrays in their order",
			scope: Scope{Pairs: []Pair{
				{TargetID: second, TargetName: "b", StageID: "prod"},
				{TargetID: first, TargetName: "a", StageID: "dev"},
				{TargetID: first, TargetName: "a", StageID: "prod"},
			}},
			assert: func(t *testing.T, args database.ListTargetStatusesParams) {
				require.Equal(t, []uuid.UUID{second, first, first}, args.TargetIds)
				require.Equal(t, []string{"b", "a", "a"}, args.TargetNames)
				require.Equal(t, []string{"prod", "dev", "prod"}, args.StageIds)
			},
		},
		{
			name: "a Stage's Freight is sorted and joined, and a Stage with none is left out",
			scope: Scope{StageFreight: map[string][]string{
				"prod":  {"b", "a", "B"},
				"dev":   {},
				"stage": nil,
			}},
			assert: func(t *testing.T, args database.ListTargetStatusesParams) {
				require.Equal(t, []string{"prod"}, args.FreightStageIds)
				// Byte order, as the query sorts what a Target is on.
				require.Equal(t, []string{"B,a,b"}, args.StageFreight)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.assert(t, params(testCase.scope))
		})
	}

	t.Run("the caller's Freight is not reordered", func(t *testing.T) {
		t.Parallel()
		names := []string{"b", "a"}
		params(Scope{StageFreight: map[string][]string{"prod": names}})
		require.Equal(t, []string{"b", "a"}, names)
	})
}

func TestEntry(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	created := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	finished := created.Add(time.Minute)
	testCases := []struct {
		name   string
		row    database.ListTargetStatusesRow
		assert func(*testing.T, Entry)
	}{
		{
			name: "a Target nothing was promoted to has no Promotion and no Freight",
			row: database.ListTargetStatusesRow{
				TargetID: id,
				StageID:  "prod",
				State:    "NeverPromoted",
				// The query reports the phase of a missing Promotion as
				// Pending; without a creation time there is none.
				LastPhase: "Pending",
			},
			assert: func(t *testing.T, entry Entry) {
				require.Equal(
					t,
					Entry{TargetID: id, StageID: "prod", State: StateNeverPromoted, Freight: []Freight{}},
					entry,
				)
			},
		},
		{
			name: "the last Promotion and the Freight are presented",
			row: database.ListTargetStatusesRow{
				TargetID:          id,
				StageID:           "prod",
				State:             "NotAttempted",
				Behind:            1,
				LastName:          "prod.abc",
				LastPhase:         "Succeeded",
				LastFreightName:   "v1",
				LastCreatedAt:     &created,
				LastFinishedAt:    &finished,
				LastRolloutNumber: 3,
				OriginKinds:       []string{"Warehouse", "Warehouse"},
				OriginNames:       []string{"app", "config"},
				FreightNames:      []string{"v1", "c1"},
				RolloutNumber:     4,
			},
			assert: func(t *testing.T, entry Entry) {
				require.Equal(
					t,
					Entry{
						TargetID: id,
						StageID:  "prod",
						State:    StateNotAttempted,
						Behind:   true,
						Last: &Promotion{
							Name:       "prod.abc",
							Freight:    "v1",
							Phase:      "Succeeded",
							Rollout:    3,
							CreatedAt:  created,
							FinishedAt: &finished,
						},
						Freight: []Freight{
							{OriginKind: "Warehouse", OriginName: "app", Name: "v1"},
							{OriginKind: "Warehouse", OriginName: "config", Name: "c1"},
						},
						Rollout: 4,
					},
					entry,
				)
			},
		},
		{
			name: "what is in flight is reported as the query found it",
			row: database.ListTargetStatusesRow{
				State:         "Promoting",
				InFlight:      true,
				LastName:      "prod.abc",
				LastPhase:     "Running",
				LastCreatedAt: &created,
			},
			assert: func(t *testing.T, entry Entry) {
				require.Equal(t, StatePromoting, entry.State)
				require.True(t, entry.InFlight)
				require.False(t, entry.Behind)
				require.Nil(t, entry.Last.FinishedAt)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.assert(t, entry(testCase.row))
		})
	}
}

func TestZeroes(t *testing.T) {
	t.Parallel()
	counts := zeroes()
	require.Len(t, counts, len(States()))
	for _, state := range States() {
		count, ok := counts[state]
		require.True(t, ok, state)
		require.Zero(t, count)
	}
}
