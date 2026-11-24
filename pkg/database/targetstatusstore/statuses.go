package targetstatusstore

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/akuity/kargo/pkg/database"
)

// Pair is a Target and a Stage that governs it. Which Stages govern which
// Targets is not recorded in the database; the caller works it out from the
// Stages' selectors and asks about the pairs it finds.
type Pair struct {
	TargetID   uuid.UUID
	TargetName string
	// StageID is the Stage's id, its Kubernetes UID, which the rows of its
	// Promotions and PromotionRollouts carry.
	StageID string
}

// Scope is what a reader asks about: the pairs, and what each Stage is
// currently on, without which no Target can be found behind it.
type Scope struct {
	// Pairs are the Targets and the Stages that govern them, in the order
	// their entries are wanted in.
	Pairs []Pair
	// StageFreight names the Freight in each Stage's current collection, by
	// the Stage's id. A Stage with none is absent, and none of its Targets
	// is behind.
	StageFreight map[string][]string
}

// Freight is one origin's Freight on a Target.
type Freight struct {
	OriginKind string
	OriginName string
	Name       string
}

// Entry is where one Target stands with one Stage that governs it.
type Entry struct {
	TargetID uuid.UUID
	StageID  string
	State    State
	// InFlight is whether a promotion to the Target is on its way or under
	// way: the State is StatePromoting or StatePending.
	InFlight bool
	// Behind is whether the Freight the Target is on is not the Stage's
	// current Freight. It holds in more States than StateBehind: a Target
	// whose last Promotion failed is usually behind as well.
	Behind bool
	// Last is the newest Promotion to the Target for the Stage, whatever its
	// phase, or nil if there has been none. Its Stage is left empty, since
	// the entry names it.
	Last *Promotion
	// Freight is what the Target is on for the Stage, one piece per origin
	// in origin order: what its newest successful Promotion left there.
	// Empty until one has succeeded.
	Freight []Freight
	// Rollout is the number of the PromotionRollout a StatePending or
	// StateNotAttempted comes from when no Promotion says it: the one the
	// Target is waiting for, or the one that never attempted it. Zero
	// otherwise.
	Rollout int64
}

// Counts tallies Targets by State.
type Counts struct {
	// States holds every State, including those nothing is in.
	States map[State]int64
	// Behind is how many of those counted are behind, whatever their State.
	Behind int64
}

// Summary tallies the entries of a Scope.
type Summary struct {
	// Targets counts each Target once, under the first State in States that
	// any of its Stages reports, and as behind if it is behind for any.
	Targets Counts
	// Stages counts each Stage's entries, by the Stage's id. A Stage with no
	// pairs in the Scope is absent.
	Stages map[string]Counts
}

// Targets returns the id, name and labels of every Target in the named
// Project, in byte order of name: what governance is worked out from. A
// Project with no Targets, or none at all, yields an empty list.
func (s *store) Targets(
	ctx context.Context,
	project string,
) ([]database.ListTargetLabelsRow, error) {
	rows, err := s.queries.ListTargetLabels(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("error listing Targets in Project %q: %w", project, err)
	}
	return rows, nil
}

// List returns the entries of the Scope's pairs, in the pairs' order: at
// most limit of them, DefaultPageSize if it is not positive, and only those
// in one of the given States when any are given. A caller pages by asking
// again with the pairs that follow the last entry returned.
func (s *store) List(
	ctx context.Context,
	scope Scope,
	states []State,
	limit int32,
) ([]Entry, error) {
	if limit <= 0 {
		limit = DefaultPageSize
	}
	pairs := scope.Pairs
	// With no States to tell apart, the page is the first pairs, so the
	// rest need not be read.
	if len(states) == 0 && len(pairs) > int(limit) {
		pairs = pairs[:limit]
	}
	if len(pairs) == 0 {
		return []Entry{}, nil
	}
	args := params(Scope{Pairs: pairs, StageFreight: scope.StageFreight})
	args.PageSize = limit
	for _, state := range states {
		args.States = append(args.States, string(state))
	}
	rows, err := s.queries.ListTargetStatuses(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("error listing Target statuses: %w", err)
	}
	entries := make([]Entry, len(rows))
	for i, row := range rows {
		entries[i] = entry(row)
	}
	return entries, nil
}

// Summarize tallies the entries of the Scope's pairs by State: each Stage's,
// and the Targets' own. The database does the counting; no entry is read
// here.
func (s *store) Summarize(ctx context.Context, scope Scope) (Summary, error) {
	summary := Summary{
		Targets: Counts{States: zeroes()},
		Stages:  map[string]Counts{},
	}
	if len(scope.Pairs) == 0 {
		return summary, nil
	}
	args := params(scope)
	args.Summarize = true
	rows, err := s.queries.ListTargetStatuses(ctx, args)
	if err != nil {
		return Summary{}, fmt.Errorf("error summarizing Target statuses: %w", err)
	}
	for _, row := range rows {
		counts := summary.Targets
		if row.StageID != "" {
			var ok bool
			if counts, ok = summary.Stages[row.StageID]; !ok {
				counts = Counts{States: zeroes()}
			}
		}
		counts.States[State(row.State)] += row.Total
		counts.Behind += row.Behind
		if row.StageID == "" {
			summary.Targets = counts
			continue
		}
		summary.Stages[row.StageID] = counts
	}
	return summary, nil
}

// params encodes a Scope for the query: its pairs as three arrays of equal
// length, and each Stage's current Freight as the names sorted in byte
// order and joined with commas, which is how the query spells the Freight a
// Target is on to compare the two.
func params(scope Scope) database.ListTargetStatusesParams {
	args := database.ListTargetStatusesParams{
		TargetIds:       make([]uuid.UUID, len(scope.Pairs)),
		TargetNames:     make([]string, len(scope.Pairs)),
		StageIds:        make([]string, len(scope.Pairs)),
		FreightStageIds: make([]string, 0, len(scope.StageFreight)),
		StageFreight:    make([]string, 0, len(scope.StageFreight)),
		States:          []string{},
	}
	for i, pair := range scope.Pairs {
		args.TargetIds[i] = pair.TargetID
		args.TargetNames[i] = pair.TargetName
		args.StageIds[i] = pair.StageID
	}
	for stage, names := range scope.StageFreight {
		if len(names) == 0 {
			continue
		}
		sorted := slices.Clone(names)
		slices.Sort(sorted)
		args.FreightStageIds = append(args.FreightStageIds, stage)
		args.StageFreight = append(args.StageFreight, strings.Join(sorted, ","))
	}
	for _, state := range States() {
		args.Precedence = append(args.Precedence, string(state))
	}
	return args
}

// entry presents a row of the query as an Entry.
func entry(row database.ListTargetStatusesRow) Entry {
	e := Entry{
		TargetID: row.TargetID,
		StageID:  row.StageID,
		State:    State(row.State),
		InFlight: row.InFlight,
		Behind:   row.Behind > 0,
		Freight:  make([]Freight, len(row.FreightNames)),
		Rollout:  row.RolloutNumber,
	}
	for i, name := range row.FreightNames {
		e.Freight[i] = Freight{
			OriginKind: row.OriginKinds[i],
			OriginName: row.OriginNames[i],
			Name:       name,
		}
	}
	if row.LastCreatedAt != nil {
		e.Last = &Promotion{
			ID:         row.LastID,
			Name:       row.LastName,
			Freight:    row.LastFreightName,
			Phase:      row.LastPhase,
			Rollout:    row.LastRolloutNumber,
			CreatedAt:  *row.LastCreatedAt,
			FinishedAt: row.LastFinishedAt,
		}
	}
	return e
}

// zeroes returns a tally with every State at zero.
func zeroes() map[State]int64 {
	counts := make(map[State]int64, len(States()))
	for _, state := range States() {
		counts[state] = 0
	}
	return counts
}
