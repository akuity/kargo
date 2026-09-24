package projects

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

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/management/dbsync/internal/syncapi"
	"github.com/akuity/kargo/pkg/database"
)

func TestDiff(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	live := testProject("live", created)
	row := database.Project{ID: "live", Name: "demo", CreatedAt: created}
	wrongName := row
	wrongName.Name = "wrong"
	wrongTimestamp := row
	wrongTimestamp.CreatedAt = created.Add(-time.Hour)
	oldSyncTime := row
	oldSyncTime.SyncedAt = created.Add(-24 * time.Hour)
	key := client.ObjectKey{Name: "demo"}
	testCases := []struct {
		name        string
		project     *kargoapi.Project
		rows        []database.Project
		listErr     error
		snapshotErr error
		incomplete  bool
		insert      bool
		want        syncapi.Changes
		wantErr     string
	}{
		{name: "empty database", project: live, want: syncapi.Changes{ToSync: []client.ObjectKey{key}}},
		{name: "matching row is not queued", project: live, rows: []database.Project{row}},
		{name: "synced_at does not cause a diff", project: live, rows: []database.Project{oldSyncTime}},
		{
			name: "different name", project: live, rows: []database.Project{wrongName},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}},
		},
		{
			name: "different creation timestamp", project: live, rows: []database.Project{wrongTimestamp},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}},
		},
		{name: "offline deletion", rows: []database.Project{row}, want: syncapi.Changes{ToDelete: []string{"live"}}},
		{
			name: "recreation", project: testProject("replacement", created), rows: []database.Project{row},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}, ToDelete: []string{"live"}},
		},
		{
			name: "insert after snapshot is never a candidate", project: live, rows: []database.Project{row}, insert: true,
		},
		{name: "failed database snapshot", snapshotErr: errors.New("offline"), wantErr: "listing mirrored projects"},
		{
			name: "failed Kubernetes list", rows: []database.Project{row},
			listErr: errors.New("offline"), wantErr: "listing objects",
		},
		{name: "incomplete list", rows: []database.Project{row}, incomplete: true, wantErr: "incomplete Kubernetes list"},
		{
			name: "invalid identity", project: testProject("", created),
			rows: []database.Project{row}, wantErr: "has no UID",
		},
		{
			name: "invalid timestamp", project: testProject("live", time.Time{}),
			rows: []database.Project{row}, wantErr: "incomplete identity",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))
			builder := fake.NewClientBuilder().WithScheme(scheme)
			if testCase.project != nil {
				builder.WithObjects(testCase.project.DeepCopy())
			}
			store := &fakeStore{rows: slices.Clone(testCase.rows), listErr: testCase.snapshotErr}
			reader := &diffReader{Reader: builder.Build(), err: testCase.listErr, incomplete: testCase.incomplete}
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
			require.Empty(t, store.pruned)
		})
	}
}

func TestMatchesTimestampPrecision(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 9, 23, 12, 0, 0, 123456789, time.UTC)
	project := testProject("live", created)
	// Test the matcher directly: the Kubernetes fake client's JSON round trip
	// itself removes subsecond precision from metav1.Time.
	row := database.Project{ID: "live", Name: "demo", CreatedAt: created.Truncate(time.Microsecond)}
	match, err := matches(project, row)
	require.NoError(t, err)
	require.True(t, match)
	row.CreatedAt = row.CreatedAt.Add(time.Microsecond)
	match, err = matches(project, row)
	require.NoError(t, err)
	require.False(t, match)
}

type diffReader struct {
	client.Reader
	err        error
	incomplete bool
	insertInto *fakeStore
	calls      int
}

func (r *diffReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	r.calls++
	if r.err != nil {
		return r.err
	}
	if r.insertInto != nil {
		r.insertInto.rows = append(r.insertInto.rows, database.Project{ID: "concurrent", Name: "concurrent"})
	}
	if err := r.Reader.List(ctx, list, opts...); err != nil {
		return err
	}
	if r.incomplete {
		list.SetContinue("next-page")
	}
	return nil
}
