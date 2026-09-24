package stages

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	stage := testStage("demo", "live", created)
	parent := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "demo", UID: "parent"}}
	objects := []client.Object{parent, stage}
	row := database.Stage{ID: "live", ProjectID: "parent", Name: "dev", CreatedAt: created}
	wrongName, wrongParent, wrongTimestamp, oldSyncTime := row, row, row, row
	wrongName.Name = "wrong"
	wrongParent.ProjectID = "wrong"
	wrongTimestamp.CreatedAt = created.Add(-time.Hour)
	oldSyncTime.SyncedAt = created.Add(-24 * time.Hour)
	key := client.ObjectKey{Namespace: "demo", Name: "dev"}
	testCases := []struct {
		name             string
		objects          []client.Object
		rows             []database.Stage
		snapshotErr      error
		stageErr         error
		parentErr        error
		incompleteStage  bool
		incompleteParent bool
		insert           bool
		want             syncapi.Changes
		wantErr          string
	}{
		{name: "empty database", objects: objects, want: syncapi.Changes{ToSync: []client.ObjectKey{key}}},
		{name: "matching row", objects: objects, rows: []database.Stage{row}},
		{name: "synced_at is ignored", objects: objects, rows: []database.Stage{oldSyncTime}},
		{
			name: "changed name", objects: objects, rows: []database.Stage{wrongName},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}},
		},
		{
			name: "changed project UID", objects: objects, rows: []database.Stage{wrongParent},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}},
		},
		{
			name: "changed creation timestamp", objects: objects, rows: []database.Stage{wrongTimestamp},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}},
		},
		{
			name: "missing parent retries without deleting the Stage", objects: []client.Object{stage},
			rows: []database.Stage{row, {ID: "stale"}},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}, ToDelete: []string{"stale"}},
		},
		{name: "offline deletion", rows: []database.Stage{row}, want: syncapi.Changes{ToDelete: []string{"live"}}},
		{
			name: "recreation", objects: []client.Object{parent, testStage("demo", "new", created)},
			rows: []database.Stage{row}, want: syncapi.Changes{ToSync: []client.ObjectKey{key}, ToDelete: []string{"live"}},
		},
		{name: "insert after snapshot", objects: objects, rows: []database.Stage{row}, insert: true},
		{name: "database failure", snapshotErr: errors.New("offline"), wantErr: "listing mirrored stages"},
		{
			name: "Stage list failure", rows: []database.Stage{row},
			stageErr: errors.New("offline"), wantErr: "listing objects",
		},
		{
			name: "parent list failure", rows: []database.Stage{row},
			parentErr: errors.New("offline"), wantErr: "listing stage parents",
		},
		{
			name: "incomplete Stage list", rows: []database.Stage{row},
			incompleteStage: true, wantErr: "incomplete Kubernetes list",
		},
		{
			name: "incomplete parent list", rows: []database.Stage{row},
			incompleteParent: true, wantErr: "incomplete stage parent list",
		},
		{
			name: "invalid identity", objects: []client.Object{parent, testStage("demo", "", created)},
			rows: []database.Stage{row}, wantErr: "has no UID",
		},
		{
			name: "invalid timestamp", objects: []client.Object{parent, testStage("demo", "live", time.Time{})},
			rows: []database.Stage{row}, wantErr: "incomplete identity",
		},
		{
			name: "same name in different Projects", rows: []database.Stage{row},
			objects: []client.Object{parent, stage, testStage("other", "other-stage", created),
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "other", UID: "other-parent"}}},
			want: syncapi.Changes{ToSync: []client.ObjectKey{{Namespace: "other", Name: "dev"}}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))
			builder := fake.NewClientBuilder().WithScheme(scheme)
			for _, obj := range testCase.objects {
				builder.WithRuntimeObjects(obj.DeepCopyObject())
			}
			store := &fakeStore{rows: slices.Clone(testCase.rows), listErr: testCase.snapshotErr}
			reader := &diffReader{
				Reader: builder.Build(), stageErr: testCase.stageErr, parentErr: testCase.parentErr,
				incompleteStage: testCase.incompleteStage, incompleteParent: testCase.incompleteParent,
			}
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

type diffReader struct {
	client.Reader
	stageErr         error
	parentErr        error
	incompleteStage  bool
	incompleteParent bool
	insertInto       *fakeStore
	calls            int
}

func (r *diffReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	r.calls++
	_, isStage := list.(*kargoapi.StageList)
	if isStage && r.stageErr != nil {
		return r.stageErr
	}
	if !isStage && r.parentErr != nil {
		return r.parentErr
	}
	if r.insertInto != nil && r.calls == 1 {
		r.insertInto.rows = append(r.insertInto.rows, database.Stage{ID: "concurrent", Name: "concurrent"})
	}
	if err := r.Reader.List(ctx, list, opts...); err != nil {
		return err
	}
	if (isStage && r.incompleteStage) || (!isStage && r.incompleteParent) {
		list.SetContinue("next-page")
	}
	return nil
}
