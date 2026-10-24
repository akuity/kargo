//go:build integration && db

package stagetargets

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/databasetest"
)

func TestReconcileIntegration(t *testing.T) {
	pool, _ := databasetest.IsolatedDatabase(t)
	queries := database.New(pool)
	r := &stageTargetsReconciler{store: queries}
	feed := newChangeFeed(queries)
	ctx := context.Background()
	created := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

	target := func(project, name, region string) database.TargetRow {
		row, err := queries.CreateTarget(ctx, database.CreateTargetParams{
			ProjectName:   project,
			Name:          name,
			Labels:        []byte(`{"region":"` + region + `"}`),
			Params:        []byte(`{}`),
			TargetSetName: "fleet",
		})
		require.NoError(t, err)
		return row
	}
	stage := func(project, name, selectors string) {
		params := database.UpsertStageParams{
			ID: project + "/" + name, ProjectName: project, Name: name, CreatedAt: created,
		}
		if selectors != "" {
			params.TargetSelectors = []byte(selectors)
		}
		require.NoError(t, queries.UpsertStage(ctx, params))
	}
	governance := func(project string) map[string][]uuid.UUID {
		rows, err := queries.ListStageTargets(ctx, project)
		require.NoError(t, err)
		out := map[string][]uuid.UUID{}
		for _, row := range rows {
			out[row.StageID] = append(out[row.StageID], row.TargetID)
		}
		return out
	}
	reconcile := func(project string) {
		_, err := r.reconcile(ctx, project)
		require.NoError(t, err)
	}
	changed := func() []string {
		projects, err := feed.changed(ctx)
		require.NoError(t, err)
		return projects
	}

	us := target("demo", "us", "us")
	eu := target("demo", "eu", "eu")
	other := target("other", "us", "us")
	stage("demo", "fleet", `[{"matchLabels":{"region":"us"}}]`)
	stage("demo", "classic", "")
	stage("other", "fleet", `[{}]`)
	require.Equal(t, []string{"demo", "other"}, changed())

	reconcile("demo")
	require.Equal(t, map[string][]uuid.UUID{"demo/fleet": {us.ID}}, governance("demo"))
	reconcile("other")
	require.Equal(t, map[string][]uuid.UUID{"other/fleet": {other.ID}}, governance("other"))
	require.Empty(t, changed(), "reconciling writes nothing the feed watches")

	// Relabeling a Target moves it between Stages.
	_, err := queries.UpdateTarget(ctx, database.UpdateTargetParams{
		ProjectName: "demo", Name: "eu", Labels: []byte(`{"region":"us"}`),
		Params: []byte(`{}`), TargetSetName: "fleet",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"demo"}, changed())
	reconcile("demo")
	require.ElementsMatch(t, []uuid.UUID{us.ID, eu.ID}, governance("demo")["demo/fleet"])

	// Reselecting a Stage moves Targets out of it.
	stage("demo", "fleet", `[{"matchLabels":{"region":"eu"}}]`)
	require.Equal(t, []string{"demo"}, changed())
	reconcile("demo")
	require.Empty(t, governance("demo"))

	// A Stage turned classic governs nothing; a new Target is picked up.
	stage("demo", "fleet", `[{}]`)
	reconcile("demo")
	require.Len(t, governance("demo")["demo/fleet"], 2)
	target("demo", "ap", "ap")
	require.Equal(t, []string{"demo"}, changed())
	reconcile("demo")
	require.Len(t, governance("demo")["demo/fleet"], 3)
	stage("demo", "fleet", "")
	require.Equal(t, []string{"demo"}, changed())
	reconcile("demo")
	require.Empty(t, governance("demo"))

	// Rows go with their Target and their Stage.
	stage("demo", "fleet", `[{}]`)
	reconcile("demo")
	_, err = queries.DeleteTarget(ctx, database.DeleteTargetParams{ProjectName: "demo", Name: "ap"})
	require.NoError(t, err)
	require.Len(t, governance("demo")["demo/fleet"], 2)
	require.Equal(t, []string{"demo"}, changed(), "a deleted Target changes the count")
	require.NoError(t, queries.DeleteStageByName(ctx, database.DeleteStageByNameParams{
		ProjectName: "demo", Name: "fleet",
	}))
	require.Empty(t, governance("demo"))
	require.Equal(t, []string{"demo"}, changed())

	// Other Projects are never touched.
	require.Equal(t, map[string][]uuid.UUID{"other/fleet": {other.ID}}, governance("other"))
}

func TestInsertSkipsVanishedRowsIntegration(t *testing.T) {
	pool, _ := databasetest.IsolatedDatabase(t)
	queries := database.New(pool)
	ctx := context.Background()
	require.NoError(t, queries.InsertStageTargets(ctx, database.InsertStageTargetsParams{
		StageIds:  []string{"no-such-stage"},
		TargetIds: []uuid.UUID{uuid.New()},
	}))
	rows, err := queries.ListStageTargets(ctx, "demo")
	require.NoError(t, err)
	require.Empty(t, rows)
}
