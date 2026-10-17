package stages

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

func TestDiff(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	live := testStage("live", created)
	row := database.StageRow{ID: "live", ProjectName: "demo", Name: "dev", CreatedAt: created}
	wrongProject := row
	wrongProject.ProjectName = "wrong"
	wrongName := row
	wrongName.Name = "wrong"
	wrongTimestamp := row
	wrongTimestamp.CreatedAt = created.Add(-time.Hour)
	oldSyncTime := row
	oldSyncTime.SyncedAt = created.Add(-24 * time.Hour)
	key := []reconcile.Request{{NamespacedName: client.ObjectKey{Namespace: "demo", Name: "dev"}}}
	testCases := []struct {
		name        string
		stage       *kargoapi.Stage
		rows        []database.StageRow
		listErr     error
		snapshotErr error
		insert      bool
		want        []reconcile.Request
		wantErr     string
	}{
		{name: "empty database", stage: live, want: key},
		{name: "matching row is not queued", stage: live, rows: []database.StageRow{row}},
		{name: "synced_at does not cause a diff", stage: live, rows: []database.StageRow{oldSyncTime}},
		{name: "different project", stage: live, rows: []database.StageRow{wrongProject}, want: key},
		{name: "different name", stage: live, rows: []database.StageRow{wrongName}, want: key},
		{
			name: "different creation timestamp", stage: live, rows: []database.StageRow{wrongTimestamp},
			want: key,
		},
		// The orphaned row is returned by project and name; reconciling it
		// finds no Stage and deletes the row.
		{name: "offline deletion", rows: []database.StageRow{row}, want: key},
		// The live Stage and the orphaned row share a key, so one request
		// covers both: the upsert replaces the old row.
		{name: "recreation", stage: testStage("replacement", created), rows: []database.StageRow{row}, want: key},
		{
			name: "insert after snapshot is never a candidate", stage: live,
			rows: []database.StageRow{row}, insert: true,
		},
		{name: "failed database snapshot", snapshotErr: errors.New("offline"), wantErr: "listing mirrored stages"},
		{
			name: "failed Kubernetes list", rows: []database.StageRow{row},
			listErr: errors.New("offline"), wantErr: "listing objects",
		},
		{
			name: "invalid identity", stage: testStage("", created),
			rows: []database.StageRow{row}, wantErr: "has no UID",
		},
		{
			name: "invalid timestamp", stage: testStage("live", time.Time{}),
			rows: []database.StageRow{row}, wantErr: "incomplete identity",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))
			builder := fake.NewClientBuilder().WithScheme(scheme)
			if testCase.stage != nil {
				builder.WithObjects(testCase.stage.DeepCopy())
			}
			store := &fakeStore{rows: slices.Clone(testCase.rows), listErr: testCase.snapshotErr}
			reader := &diffReader{Reader: builder.Build(), err: testCase.listErr}
			if testCase.insert {
				reader.insertInto = store
			}
			changes, err := NewSyncer(reader, store).Diff(context.Background())
			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, testCase.want, changes)
			if testCase.snapshotErr != nil {
				require.Zero(t, reader.calls)
			}
			if testCase.insert {
				require.Len(t, store.rows, len(testCase.rows)+1)
			}
			require.Empty(t, store.upserts)
			require.Empty(t, store.deletes)
		})
	}
}

func TestMatchesTimestampPrecision(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 10, 15, 12, 0, 0, 123456789, time.UTC)
	stage := testStage("live", created)
	// Test the matcher directly: the Kubernetes fake client's JSON round trip
	// itself removes subsecond precision from metav1.Time.
	row := database.StageRow{
		ID: "live", ProjectName: "demo", Name: "dev", CreatedAt: created.Truncate(time.Microsecond),
	}
	match, err := matches(stage, row)
	require.NoError(t, err)
	require.True(t, match)
	row.CreatedAt = row.CreatedAt.Add(time.Microsecond)
	match, err = matches(stage, row)
	require.NoError(t, err)
	require.False(t, match)
}

type diffReader struct {
	client.Reader
	err        error
	insertInto *fakeStore
	calls      int
}

func (r *diffReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	r.calls++
	if r.err != nil {
		return r.err
	}
	if r.insertInto != nil {
		r.insertInto.rows = append(r.insertInto.rows, database.StageRow{
			ID: "concurrent", ProjectName: "demo", Name: "concurrent",
		})
	}
	return r.Reader.List(ctx, list, opts...)
}
