package stagetargets

import (
	"context"
	"fmt"
	"sync"

	"github.com/akuity/kargo/pkg/database"
)

type inputLister interface {
	ListStageTargetInputs(context.Context) ([]database.ListStageTargetInputsRow, error)
}

// changeFeed tells which Projects need their governance settled again. A
// Project's governance follows from its Stages' selectors and its Targets'
// labels, and the database condenses both, per Project, into a digest of the
// Stages and the Targets' count and latest write. The feed keeps what it saw
// last and, each time it lists, returns the Projects whose inputs differ, or
// every Project the first time. Nothing publishes changes to Stages or
// Targets, so this poll is how the controller hears of them.
type changeFeed struct {
	store inputLister
	mu    sync.Mutex
	seen  map[string]database.ListStageTargetInputsRow
}

func newChangeFeed(store inputLister) *changeFeed {
	return &changeFeed{store: store}
}

// changed returns the Projects whose inputs changed since the last call, and
// remembers what it read. A Project left with neither Stages nor Targets has
// no rows to settle and is simply forgotten.
func (f *changeFeed) changed(ctx context.Context) ([]string, error) {
	rows, err := f.store.ListStageTargetInputs(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing stage target inputs: %w", err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	next := make(map[string]database.ListStageTargetInputsRow, len(rows))
	var projects []string
	for _, row := range rows {
		next[row.ProjectName] = row
		if prev, ok := f.seen[row.ProjectName]; !ok || !sameInputs(prev, row) {
			projects = append(projects, row.ProjectName)
		}
	}
	f.seen = next
	return projects, nil
}

// all returns every Project with Stages or Targets, for the periodic resync
// that settles whatever the feed could have missed.
func (f *changeFeed) all(ctx context.Context) ([]string, error) {
	rows, err := f.store.ListStageTargetInputs(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing stage target inputs: %w", err)
	}
	projects := make([]string, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, row.ProjectName)
	}
	return projects, nil
}

func sameInputs(a, b database.ListStageTargetInputsRow) bool {
	return a.StageDigest == b.StageDigest &&
		a.TargetCount == b.TargetCount &&
		a.TargetsUpdatedAt.Equal(b.TargetsUpdatedAt)
}
