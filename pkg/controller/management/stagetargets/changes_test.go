package stagetargets

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database"
)

type fakeInputs struct {
	rows []database.ListStageTargetInputsRow
	err  error
}

func (f *fakeInputs) ListStageTargetInputs(context.Context) ([]database.ListStageTargetInputsRow, error) {
	return f.rows, f.err
}

func TestChangeFeed(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	input := func(project string) database.ListStageTargetInputsRow {
		return database.ListStageTargetInputsRow{
			ProjectName: project, StageDigest: "d", TargetCount: 2, TargetsUpdatedAt: at,
		}
	}
	ctx := context.Background()
	store := &fakeInputs{rows: []database.ListStageTargetInputsRow{input("a"), input("b"), input("c"), input("d")}}
	feed := newChangeFeed(store)

	projects, err := feed.changed(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b", "c", "d"}, projects, "everything is new the first time")

	projects, err = feed.changed(ctx)
	require.NoError(t, err)
	require.Empty(t, projects, "nothing changed")

	reselected, recounted, rewritten := input("a"), input("b"), input("c")
	reselected.StageDigest = "e"
	recounted.TargetCount = 3
	rewritten.TargetsUpdatedAt = at.Add(time.Second)
	// d is gone; e is new.
	store.rows = []database.ListStageTargetInputsRow{reselected, recounted, rewritten, input("e")}
	projects, err = feed.changed(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b", "c", "e"}, projects)

	projects, err = feed.all(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b", "c", "e"}, projects)

	store.err = errors.New("offline")
	_, err = feed.changed(ctx)
	require.ErrorContains(t, err, "offline")
	_, err = feed.all(ctx)
	require.ErrorContains(t, err, "offline")
	// A failed list forgets nothing: what was seen still holds.
	store.err = nil
	projects, err = feed.changed(ctx)
	require.NoError(t, err)
	require.Empty(t, projects)
}
