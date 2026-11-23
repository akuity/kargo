//go:build integration && db

package targetstatusstore

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database/databasetest"
)

// governing records a target-aware Stage and returns its id.
func (f *fixture) governing(name string) string {
	f.t.Helper()
	id := f.project + "/" + name
	f.stage(id, name)
	return id
}

// pair returns the named Target, created if need be, with the Stage.
func (f *fixture) pair(name, stageID string) Pair {
	f.t.Helper()
	return Pair{TargetID: f.target(name), TargetName: name, StageID: stageID}
}

// rolloutTo records a PromotionRollout of the Stage to the pairs' Targets,
// in the given phase.
func (f *fixture) rolloutTo(stageName, phase string, pairs ...Pair) rollout {
	f.t.Helper()
	ids := make([]uuid.UUID, len(pairs))
	for i, pair := range pairs {
		ids[i] = pair.TargetID
	}
	return f.rollout(stageName, phase, ids...)
}

// promote records a Promotion of the Stage to the pair's Target, in the
// given phase, as part of the rollout if one is given. A successful one
// leaves the named Freight on the Target.
func (f *fixture) promote(stageName string, pair Pair, phase string, r *rollout, freight ...string) string {
	f.t.Helper()
	return f.promotion(stageName, pair.TargetName, phase, r, freight...)
}

// entry reads the one entry of the pair, with the Stage on the given
// Freight.
func (f *fixture) entry(pair Pair, stageFreight ...string) Entry {
	f.t.Helper()
	entries, err := f.store.List(
		context.Background(),
		Scope{Pairs: []Pair{pair}, StageFreight: map[string][]string{pair.StageID: stageFreight}},
		nil,
		0,
	)
	require.NoError(f.t, err)
	require.Len(f.t, entries, 1)
	require.Equal(f.t, pair.TargetID, entries[0].TargetID)
	require.Equal(f.t, pair.StageID, entries[0].StageID)
	return entries[0]
}

func app(name string) Freight {
	return Freight{OriginKind: "Warehouse", OriginName: "app", Name: name}
}

func config(name string) Freight {
	return Freight{OriginKind: "Warehouse", OriginName: "config", Name: name}
}

// TestStateIntegration walks the derivation of a State from top to bottom.
// Each case has a Stage of its own, so none sees another's history.
func TestStateIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := databasetest.IsolatedDatabase(t)
	testCases := []struct {
		name string
		run  func(*testing.T, *fixture, string, string)
	}{
		{
			name: "a Target nothing has been promoted to has never been promoted",
			run: func(t *testing.T, f *fixture, _, stageID string) {
				entry := f.entry(f.pair("a", stageID), "v2")
				require.Equal(t, StateNeverPromoted, entry.State)
				require.False(t, entry.InFlight)
				require.False(t, entry.Behind)
				require.Nil(t, entry.Last)
				require.Empty(t, entry.Freight)
				require.Zero(t, entry.Rollout)
			},
		},
		{
			name: "a running Promotion is promoting",
			run: func(t *testing.T, f *fixture, stageName, stageID string) {
				a := f.pair("a", stageID)
				r := f.rolloutTo(stageName, "Running", a)
				name := f.promote(stageName, a, "Running", &r, "v2")
				entry := f.entry(a, "v2")
				require.Equal(t, StatePromoting, entry.State)
				require.True(t, entry.InFlight)
				require.False(t, entry.Behind)
				require.NotNil(t, entry.Last)
				require.Equal(t, name, entry.Last.Name)
				require.Equal(t, "Running", entry.Last.Phase)
				require.Equal(t, "v2", entry.Last.Freight)
				require.Equal(t, r.number, entry.Last.Rollout)
				require.Nil(t, entry.Last.FinishedAt)
				require.Empty(t, entry.Freight)
				require.Zero(t, entry.Rollout)
			},
		},
		{
			name: "a Promotion that has yet to start is pending",
			run: func(t *testing.T, f *fixture, stageName, stageID string) {
				a, b := f.pair("a", stageID), f.pair("b", stageID)
				f.promote(stageName, a, "Pending", nil, "v2")
				// One the controller has not picked up has no phase at all.
				f.promote(stageName, b, "", nil, "v2")
				for _, pair := range []Pair{a, b} {
					entry := f.entry(pair, "v2")
					require.Equal(t, StatePending, entry.State)
					require.True(t, entry.InFlight)
					require.Equal(t, "Pending", entry.Last.Phase)
					require.Zero(t, entry.Rollout)
				}
			},
		},
		{
			name: "a Promotion that succeeded leaves its Freight, and the Target up to date",
			run: func(t *testing.T, f *fixture, stageName, stageID string) {
				a := f.pair("a", stageID)
				f.promote(stageName, a, "Succeeded", nil, "v1", "c1")
				name := f.promote(stageName, a, "Succeeded", nil, "v2", "c1")
				// The Stage's Freight is compared whatever order it is given in.
				entry := f.entry(a, "v2", "c1")
				require.Equal(t, StateUpToDate, entry.State)
				require.False(t, entry.InFlight)
				require.False(t, entry.Behind)
				require.Equal(t, name, entry.Last.Name)
				require.Equal(t, "Succeeded", entry.Last.Phase)
				require.Zero(t, entry.Last.Rollout)
				require.NotNil(t, entry.Last.FinishedAt)
				require.Equal(t, []Freight{app("v2"), config("c1")}, entry.Freight)
			},
		},
		{
			name: "a Target on other Freight than its Stage's is behind",
			run: func(t *testing.T, f *fixture, stageName, stageID string) {
				a := f.pair("a", stageID)
				f.promote(stageName, a, "Succeeded", nil, "v1", "c1")
				for _, stageFreight := range [][]string{
					{"v2", "c1"}, // one origin has moved on
					{"v1"},       // the Stage has dropped an origin
					{"v1", "c1", "extra"},
				} {
					entry := f.entry(a, stageFreight...)
					require.Equal(t, StateBehind, entry.State, stageFreight)
					require.True(t, entry.Behind)
					require.False(t, entry.InFlight)
					require.Equal(t, []Freight{app("v1"), config("c1")}, entry.Freight)
				}
			},
		},
		{
			name: "no Target is behind a Stage with no current Freight",
			run: func(t *testing.T, f *fixture, stageName, stageID string) {
				a := f.pair("a", stageID)
				f.promote(stageName, a, "Succeeded", nil, "v1")
				entry := f.entry(a)
				require.Equal(t, StateUpToDate, entry.State)
				require.False(t, entry.Behind)
				require.Equal(t, []Freight{app("v1")}, entry.Freight)
			},
		},
		{
			name: "a Promotion that did not succeed has failed, and leaves the Freight that was there",
			run: func(t *testing.T, f *fixture, stageName, stageID string) {
				for _, phase := range []string{"Failed", "Errored", "Aborted"} {
					pair := f.pair("target-"+phase, stageID)
					f.promote(stageName, pair, "Succeeded", nil, "v1")
					f.promote(stageName, pair, phase, nil, "v2")
					entry := f.entry(pair, "v2")
					require.Equal(t, StateFailed, entry.State, phase)
					require.False(t, entry.InFlight)
					require.True(t, entry.Behind)
					require.Equal(t, phase, entry.Last.Phase)
					require.Equal(t, "v2", entry.Last.Freight)
					require.Equal(t, []Freight{app("v1")}, entry.Freight)
					// It is no longer behind once the Stage is back on what it has.
					require.False(t, f.entry(pair, "v1").Behind)
				}
				// One that has never succeeded is on nothing, and behind nothing.
				never := f.pair("never", stageID)
				f.promote(stageName, never, "Failed", nil, "v2")
				entry := f.entry(never, "v2")
				require.Equal(t, StateFailed, entry.State)
				require.False(t, entry.Behind)
				require.Empty(t, entry.Freight)
			},
		},
		{
			name: "a Target an unfinished rollout has not reached is pending, not unattempted",
			run: func(t *testing.T, f *fixture, stageName, stageID string) {
				reached, waiting, fresh := f.pair("a", stageID), f.pair("b", stageID), f.pair("c", stageID)
				first := f.rolloutTo(stageName, "Succeeded", reached, waiting)
				f.promote(stageName, reached, "Succeeded", &first, "v1")
				old := f.promote(stageName, waiting, "Succeeded", &first, "v1")
				running := f.rolloutTo(stageName, "Running", reached, waiting, fresh)
				f.promote(stageName, reached, "Succeeded", &running, "v2")

				// The rollout has promoted to this one: its Promotion speaks.
				entry := f.entry(reached, "v2")
				require.Equal(t, StateUpToDate, entry.State)
				require.False(t, entry.InFlight)
				require.Zero(t, entry.Rollout)

				// It has yet to reach this one, which is still on the old Freight.
				entry = f.entry(waiting, "v2")
				require.Equal(t, StatePending, entry.State)
				require.True(t, entry.InFlight)
				require.True(t, entry.Behind)
				require.Equal(t, running.number, entry.Rollout)
				require.Equal(t, old, entry.Last.Name)
				require.Equal(t, []Freight{app("v1")}, entry.Freight)

				// Nor this one, which nothing has ever been promoted to.
				entry = f.entry(fresh, "v2")
				require.Equal(t, StatePending, entry.State)
				require.True(t, entry.InFlight)
				require.Equal(t, running.number, entry.Rollout)
				require.Nil(t, entry.Last)

				// A rollout that has not started yet is unfinished too.
				f.phase(running, "Pending")
				require.Equal(t, StatePending, f.entry(waiting, "v2").State)
			},
		},
		{
			name: "a Target a rollout ended without attempting was not attempted",
			run: func(t *testing.T, f *fixture, stageName, stageID string) {
				for _, phase := range []string{"Failed", "Errored", "Aborted"} {
					promoted := f.pair("promoted-"+phase, stageID)
					never := f.pair("never-"+phase, stageID)
					first := f.rolloutTo(stageName, "Succeeded", promoted)
					old := f.promote(stageName, promoted, "Succeeded", &first, "v1")
					ended := f.rolloutTo(stageName, phase, promoted, never)

					entry := f.entry(promoted, "v2")
					require.Equal(t, StateNotAttempted, entry.State, phase)
					require.False(t, entry.InFlight)
					require.True(t, entry.Behind)
					require.Equal(t, ended.number, entry.Rollout)
					require.Equal(t, old, entry.Last.Name)
					require.Equal(t, []Freight{app("v1")}, entry.Freight)

					entry = f.entry(never, "v2")
					require.Equal(t, StateNotAttempted, entry.State, phase)
					require.Equal(t, ended.number, entry.Rollout)
					require.Nil(t, entry.Last)
					require.False(t, entry.Behind)
				}
			},
		},
		{
			name: "a rollout that succeeded is never read as not having attempted a Target",
			run: func(t *testing.T, f *fixture, stageName, stageID string) {
				// The rollout is done but the mirror has yet to record one of
				// its Promotions.
				a := f.pair("a", stageID)
				first := f.rolloutTo(stageName, "Succeeded", a)
				f.promote(stageName, a, "Succeeded", &first, "v1")
				f.rolloutTo(stageName, "Succeeded", a)
				entry := f.entry(a, "v2")
				require.Equal(t, StateBehind, entry.State)
				require.Zero(t, entry.Rollout)
			},
		},
		{
			name: "a later rollout that names only some Targets leaves the others as they were",
			run: func(t *testing.T, f *fixture, stageName, stageID string) {
				failed := f.pair("failed", stageID)
				succeeded := f.pair("succeeded", stageID)
				unreached := f.pair("unreached", stageID)
				first := f.rolloutTo(stageName, "Succeeded", failed, succeeded, unreached)
				for _, pair := range []Pair{failed, succeeded, unreached} {
					f.promote(stageName, pair, "Succeeded", &first, "v1")
				}
				// The second rollout fails on one Target and stops before another.
				second := f.rolloutTo(stageName, "Failed", failed, succeeded, unreached)
				f.promote(stageName, succeeded, "Succeeded", &second, "v2")
				f.promote(stageName, failed, "Failed", &second, "v2")
				before := map[string]Entry{}
				for _, pair := range []Pair{failed, succeeded, unreached} {
					before[pair.TargetName] = f.entry(pair, "v2")
				}
				require.Equal(t, StateFailed, before["failed"].State)
				require.Equal(t, StateUpToDate, before["succeeded"].State)
				require.Equal(t, StateNotAttempted, before["unreached"].State)
				require.Equal(t, second.number, before["unreached"].Rollout)

				// A retry names only the Target that failed. While it has not
				// reached it, that Target is pending and the others unchanged.
				retry := f.rolloutTo(stageName, "Running", failed)
				entry := f.entry(failed, "v2")
				require.Equal(t, StatePending, entry.State)
				require.Equal(t, retry.number, entry.Rollout)
				require.Equal(t, before["succeeded"], f.entry(succeeded, "v2"))
				require.Equal(t, before["unreached"], f.entry(unreached, "v2"))

				// Nor does the retry finishing change them, however it ends.
				f.promote(stageName, failed, "Succeeded", &retry, "v2")
				for _, phase := range []string{"Succeeded", "Failed"} {
					f.phase(retry, phase)
					require.Equal(t, StateUpToDate, f.entry(failed, "v2").State)
					require.Equal(t, before["succeeded"], f.entry(succeeded, "v2"))
					require.Equal(t, before["unreached"], f.entry(unreached, "v2"))
				}

				// A retry that names the unreached Target and ends before
				// attempting it leaves it unattempted, now by the retry.
				again := f.rolloutTo(stageName, "Aborted", unreached)
				entry = f.entry(unreached, "v2")
				require.Equal(t, StateNotAttempted, entry.State)
				require.Equal(t, again.number, entry.Rollout)
			},
		},
		{
			name: "a rollout still to reach a Target outweighs one that never attempted it",
			run: func(t *testing.T, f *fixture, stageName, stageID string) {
				a := f.pair("a", stageID)
				waiting := f.rolloutTo(stageName, "Running", a)
				// A later rollout, queued behind it, was aborted before it started.
				f.rolloutTo(stageName, "Aborted", a)
				entry := f.entry(a, "v2")
				require.Equal(t, StatePending, entry.State)
				require.True(t, entry.InFlight)
				require.Equal(t, waiting.number, entry.Rollout)
			},
		},
		{
			name: "a Promotion in flight outweighs the rollouts queued behind it",
			run: func(t *testing.T, f *fixture, stageName, stageID string) {
				a, b := f.pair("a", stageID), f.pair("b", stageID)
				running := f.rolloutTo(stageName, "Running", a, b)
				f.promote(stageName, a, "Running", &running, "v2")
				f.promote(stageName, b, "Pending", &running, "v2")
				f.rolloutTo(stageName, "Pending", a, b)
				entry := f.entry(a, "v2")
				require.Equal(t, StatePromoting, entry.State)
				require.Zero(t, entry.Rollout)
				entry = f.entry(b, "v2")
				require.Equal(t, StatePending, entry.State)
				require.Zero(t, entry.Rollout)
			},
		},
		{
			name: "a Promotion created on its own is weighed against rollouts by time",
			run: func(t *testing.T, f *fixture, stageName, stageID string) {
				a := f.pair("a", stageID)
				// This rollout ended without attempting the Target, but the
				// Promotion came after it and is the later word.
				f.rolloutTo(stageName, "Failed", a)
				f.promote(stageName, a, "Succeeded", nil, "v1")
				entry := f.entry(a, "v1")
				require.Equal(t, StateUpToDate, entry.State)
				require.Zero(t, entry.Rollout)
				// One that comes after the Promotion is.
				later := f.rolloutTo(stageName, "Failed", a)
				entry = f.entry(a, "v1")
				require.Equal(t, StateNotAttempted, entry.State)
				require.Equal(t, later.number, entry.Rollout)
			},
		},
		{
			name: "only the Stage's own Promotions and rollouts count",
			run: func(t *testing.T, f *fixture, stageName, stageID string) {
				a := f.pair("a", stageID)
				otherName := stageName + "-other"
				otherID := f.governing(otherName)
				r := f.rolloutTo(otherName, "Running", a)
				f.promote(otherName, a, "Succeeded", &r, "v1")
				f.rolloutTo(otherName, "Failed", a)
				require.Equal(t, StateNeverPromoted, f.entry(a, "v1").State)
				// The same Target is elsewhere with the other Stage.
				other := Pair{TargetID: a.TargetID, TargetName: a.TargetName, StageID: otherID}
				require.Equal(t, StateNotAttempted, f.entry(other, "v1").State)
				// A Stage the database has no rows for governs Targets all the same.
				unknown := Pair{TargetID: a.TargetID, TargetName: a.TargetName, StageID: "no-such-stage"}
				require.Equal(t, StateNeverPromoted, f.entry(unknown, "v1").State)
			},
		},
	}
	for i, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// A Project and a Stage of its own keep each case's history apart.
			f := newFixture(t, pool, fmt.Sprintf("project-%d", i))
			testCase.run(t, f, "stage", f.governing("stage"))
		})
	}
}

// TestStatesIntegration checks that the vocabulary the query speaks is the
// one States declares: every State is derived for some Target, and nothing
// else is.
func TestStatesIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := databasetest.IsolatedDatabase(t)
	f, pairs, stageFreight := newFleet(t, pool)
	entries, err := f.store.List(
		context.Background(), Scope{Pairs: pairs, StageFreight: stageFreight}, nil, 100,
	)
	require.NoError(t, err)
	seen := map[State]bool{}
	for _, entry := range entries {
		require.True(t, entry.State.Valid(), entry.State)
		require.Equal(t, entry.State == StatePromoting || entry.State == StatePending, entry.InFlight)
		seen[entry.State] = true
	}
	for _, state := range States() {
		require.True(t, seen[state], state)
	}
}

// newFleet records a Project with two Stages and a Target in every State
// under the first of them, and returns the pairs in Target then Stage
// order with the Stages' current Freight. The second Stage governs three of
// the Targets.
func newFleet(t *testing.T, pool *pgxpool.Pool) (*fixture, []Pair, map[string][]string) {
	f := newFixture(t, pool, "fleet")
	prod, canary := f.governing("prod"), f.governing("canary")
	behind := f.pair("behind", prod)
	failed := f.pair("failed", prod)
	never := f.pair("never", prod)
	pending := f.pair("pending", prod)
	promoting := f.pair("promoting", prod)
	unattempted := f.pair("unattempted", prod)
	current := f.pair("up-to-date", prod)

	first := f.rolloutTo("prod", "Succeeded", behind, failed, pending, promoting, unattempted, current)
	for _, pair := range []Pair{behind, failed, pending, promoting, unattempted, current} {
		f.promote("prod", pair, "Succeeded", &first, "v1")
	}
	second := f.rolloutTo("prod", "Failed", failed, unattempted, current)
	f.promote("prod", current, "Succeeded", &second, "v2")
	f.promote("prod", failed, "Failed", &second, "v2")
	third := f.rolloutTo("prod", "Running", pending, promoting)
	f.promote("prod", promoting, "Running", &third, "v2")

	// Under canary: one Target up to date, one that failed, one never
	// promoted to.
	canaryCurrent := Pair{TargetID: behind.TargetID, TargetName: "behind", StageID: canary}
	canaryFailed := Pair{TargetID: current.TargetID, TargetName: "up-to-date", StageID: canary}
	canaryNever := Pair{TargetID: never.TargetID, TargetName: "never", StageID: canary}
	f.promote("canary", canaryCurrent, "Succeeded", nil, "v2")
	f.promote("canary", canaryFailed, "Succeeded", nil, "v1")
	f.promote("canary", canaryFailed, "Failed", nil, "v2")

	pairs := []Pair{
		canaryCurrent, behind,
		failed,
		canaryNever, never,
		pending,
		promoting,
		unattempted,
		canaryFailed, current,
	}
	return f, pairs, map[string][]string{prod: {"v2"}, canary: {"v2"}}
}

func TestListIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := databasetest.IsolatedDatabase(t)
	f, pairs, stageFreight := newFleet(t, pool)
	ctx := context.Background()
	scope := Scope{Pairs: pairs, StageFreight: stageFreight}
	// states lists the entries' States, which also shows their order.
	states := func(entries []Entry) []State {
		out := make([]State, len(entries))
		for i, entry := range entries {
			out[i] = entry.State
		}
		return out
	}
	all := []State{
		StateUpToDate, StateBehind, // behind: canary, prod
		StateFailed,
		StateNeverPromoted, StateNeverPromoted, // never: canary, prod
		StatePending,
		StatePromoting,
		StateNotAttempted,
		StateFailed, StateUpToDate, // up-to-date: canary, prod
	}
	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "no pairs yield no entries",
			run: func(t *testing.T) {
				entries, err := f.store.List(ctx, Scope{StageFreight: stageFreight}, nil, 10)
				require.NoError(t, err)
				require.NotNil(t, entries)
				require.Empty(t, entries)
			},
		},
		{
			name: "entries come in the pairs' order, each with its own pair",
			run: func(t *testing.T) {
				entries, err := f.store.List(ctx, scope, nil, 0)
				require.NoError(t, err)
				require.Equal(t, all, states(entries))
				for i, entry := range entries {
					require.Equal(t, pairs[i].TargetID, entry.TargetID)
					require.Equal(t, pairs[i].StageID, entry.StageID)
				}
			},
		},
		{
			name: "a page is the first entries, and the next is read from the pairs after it",
			run: func(t *testing.T) {
				var read []State
				for from := 0; from < len(pairs); from += 3 {
					entries, err := f.store.List(
						ctx, Scope{Pairs: pairs[from:], StageFreight: stageFreight}, nil, 3,
					)
					require.NoError(t, err)
					require.LessOrEqual(t, len(entries), 3)
					require.Equal(t, pairs[from].TargetID, entries[0].TargetID)
					read = append(read, states(entries)...)
				}
				require.Equal(t, all, read)
			},
		},
		{
			name: "a State filter keeps only entries in the States given, in order",
			run: func(t *testing.T) {
				entries, err := f.store.List(ctx, scope, []State{StateFailed}, 0)
				require.NoError(t, err)
				require.Equal(t, []State{StateFailed, StateFailed}, states(entries))
				require.Equal(t, pairs[2].TargetID, entries[0].TargetID)
				require.Equal(t, pairs[8].TargetID, entries[1].TargetID)

				entries, err = f.store.List(
					ctx, scope, []State{StateNeverPromoted, StatePromoting, StateNotAttempted}, 0,
				)
				require.NoError(t, err)
				require.Equal(
					t,
					[]State{StateNeverPromoted, StateNeverPromoted, StatePromoting, StateNotAttempted},
					states(entries),
				)
			},
		},
		{
			name: "a filtered page is cut from the entries that match, not from the pairs",
			run: func(t *testing.T) {
				filter := []State{StateNeverPromoted, StateUpToDate}
				entries, err := f.store.List(ctx, scope, filter, 3)
				require.NoError(t, err)
				require.Equal(t, []State{StateUpToDate, StateNeverPromoted, StateNeverPromoted}, states(entries))
				// The page ended at the fifth pair; the next starts after it.
				require.Equal(t, pairs[4].TargetID, entries[2].TargetID)
				entries, err = f.store.List(
					ctx, Scope{Pairs: pairs[5:], StageFreight: stageFreight}, filter, 3,
				)
				require.NoError(t, err)
				require.Equal(t, []State{StateUpToDate}, states(entries))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.run(t)
		})
	}
}

func TestSummarizeIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := databasetest.IsolatedDatabase(t)
	f, pairs, stageFreight := newFleet(t, pool)
	ctx := context.Background()
	prod, canary := "fleet/prod", "fleet/canary"
	counts := func(behind int64, states map[State]int64) Counts {
		tally := Counts{States: zeroes(), Behind: behind}
		for state, count := range states {
			tally.States[state] = count
		}
		return tally
	}
	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "no pairs are summarized as nothing, with every State present",
			run: func(t *testing.T) {
				summary, err := f.store.Summarize(ctx, Scope{StageFreight: stageFreight})
				require.NoError(t, err)
				require.Equal(t, Summary{Targets: counts(0, nil), Stages: map[string]Counts{}}, summary)
				require.Len(t, summary.Targets.States, len(States()))
			},
		},
		{
			name: "each Stage's entries are counted by State, and each Target once under its worst",
			run: func(t *testing.T) {
				summary, err := f.store.Summarize(ctx, Scope{Pairs: pairs, StageFreight: stageFreight})
				require.NoError(t, err)
				require.Equal(
					t,
					// Every Target but the one up to date, and the one never
					// promoted to, is still on the Freight before the Stage's.
					counts(5, map[State]int64{
						StatePromoting:     1,
						StatePending:       1,
						StateFailed:        1,
						StateNotAttempted:  1,
						StateNeverPromoted: 1,
						StateBehind:        1,
						StateUpToDate:      1,
					}),
					summary.Stages[prod],
				)
				require.Equal(
					t,
					counts(1, map[State]int64{
						StateFailed:        1,
						StateNeverPromoted: 1,
						StateUpToDate:      1,
					}),
					summary.Stages[canary],
				)
				require.Len(t, summary.Stages, 2)
				// Seven Targets. The one up to date for prod failed for canary,
				// so it counts as failed, and as behind, which it is for
				// canary; the one behind prod is up to date for canary, and
				// counts as behind.
				require.Equal(
					t,
					counts(6, map[State]int64{
						StatePromoting:     1,
						StatePending:       1,
						StateFailed:        2,
						StateNotAttempted:  1,
						StateNeverPromoted: 1,
						StateBehind:        1,
					}),
					summary.Targets,
				)
			},
		},
		{
			name: "the counts are those of the entries a list returns",
			run: func(t *testing.T) {
				scope := Scope{Pairs: pairs, StageFreight: stageFreight}
				summary, err := f.store.Summarize(ctx, scope)
				require.NoError(t, err)
				entries, err := f.store.List(ctx, scope, nil, 100)
				require.NoError(t, err)
				listed := map[string]Counts{}
				for _, entry := range entries {
					tally, ok := listed[entry.StageID]
					if !ok {
						tally = counts(0, nil)
					}
					tally.States[entry.State]++
					if entry.Behind {
						tally.Behind++
					}
					listed[entry.StageID] = tally
				}
				require.Equal(t, listed, summary.Stages)
			},
		},
		{
			name: "a narrower scope counts only its own pairs",
			run: func(t *testing.T) {
				var canaryPairs []Pair
				for _, pair := range pairs {
					if pair.StageID == canary {
						canaryPairs = append(canaryPairs, pair)
					}
				}
				summary, err := f.store.Summarize(ctx, Scope{Pairs: canaryPairs, StageFreight: stageFreight})
				require.NoError(t, err)
				require.Len(t, summary.Stages, 1)
				require.Equal(t, summary.Stages[canary], summary.Targets)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.run(t)
		})
	}
}

func TestTargetsIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := databasetest.IsolatedDatabase(t)
	f := newFixture(t, pool, "fleet")
	other := newFixture(t, pool, "other")
	ctx := context.Background()
	// Byte order puts upper case before lower, whatever the database's
	// collation would say.
	for _, name := range []string{"b", "a", "Z"} {
		f.pair(name, "")
	}
	other.pair("elsewhere", "")

	rows, err := f.store.Targets(ctx, "fleet")
	require.NoError(t, err)
	names := make([]string, len(rows))
	for i, row := range rows {
		names[i] = row.Name
		require.NotEqual(t, uuid.Nil, row.ID)
		require.Equal(t, map[string]string{"region": "us"}, row.Labels)
	}
	require.Equal(t, []string{"Z", "a", "b"}, names)

	rows, err = f.store.Targets(ctx, "no-such-project")
	require.NoError(t, err)
	require.Empty(t, rows)
}
