package warehouses

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
	warehouse := testWarehouse("demo", "live", created)
	parent := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "demo", UID: "parent"}}
	objects := []client.Object{parent, warehouse}
	row := database.Warehouse{ID: "live", ProjectID: "parent", Name: "dev", CreatedAt: created}
	wrongName, wrongParent, wrongTimestamp, oldSyncTime := row, row, row, row
	wrongName.Name = "wrong"
	wrongParent.ProjectID = "wrong"
	wrongTimestamp.CreatedAt = created.Add(-time.Hour)
	oldSyncTime.SyncedAt = created.Add(-24 * time.Hour)
	key := client.ObjectKey{Namespace: "demo", Name: "dev"}
	testCases := []struct {
		name                string
		objects             []client.Object
		rows                []database.Warehouse
		snapshotErr         error
		warehouseErr        error
		parentErr           error
		incompleteWarehouse bool
		incompleteParent    bool
		insert              bool
		want                syncapi.Changes
		wantErr             string
	}{
		{name: "empty database", objects: objects, want: syncapi.Changes{ToSync: []client.ObjectKey{key}}},
		{name: "matching row", objects: objects, rows: []database.Warehouse{row}},
		{name: "synced_at is ignored", objects: objects, rows: []database.Warehouse{oldSyncTime}},
		{
			name: "changed name", objects: objects, rows: []database.Warehouse{wrongName},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}},
		},
		{
			name: "changed project UID", objects: objects, rows: []database.Warehouse{wrongParent},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}},
		},
		{
			name: "changed creation timestamp", objects: objects, rows: []database.Warehouse{wrongTimestamp},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}},
		},
		{
			name: "missing parent retries without deleting the Warehouse", objects: []client.Object{warehouse},
			rows: []database.Warehouse{row, {ID: "stale"}},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}, ToDelete: []string{"stale"}},
		},
		{name: "offline deletion", rows: []database.Warehouse{row}, want: syncapi.Changes{ToDelete: []string{"live"}}},
		{
			name: "recreation", objects: []client.Object{parent, testWarehouse("demo", "new", created)},
			rows: []database.Warehouse{row},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}, ToDelete: []string{"live"}},
		},
		{name: "insert after snapshot", objects: objects, rows: []database.Warehouse{row}, insert: true},
		{name: "database failure", snapshotErr: errors.New("offline"), wantErr: "listing mirrored warehouses"},
		{
			name: "Warehouse list failure", rows: []database.Warehouse{row},
			warehouseErr: errors.New("offline"), wantErr: "listing objects",
		},
		{
			name: "parent list failure", rows: []database.Warehouse{row},
			parentErr: errors.New("offline"), wantErr: "listing warehouse parents",
		},
		{
			name: "incomplete Warehouse list", rows: []database.Warehouse{row},
			incompleteWarehouse: true, wantErr: "incomplete Kubernetes list",
		},
		{
			name: "incomplete parent list", rows: []database.Warehouse{row},
			incompleteParent: true, wantErr: "incomplete warehouse parent list",
		},
		{
			name: "invalid identity", objects: []client.Object{parent, testWarehouse("demo", "", created)},
			rows: []database.Warehouse{row}, wantErr: "has no UID",
		},
		{
			name: "invalid timestamp", objects: []client.Object{parent, testWarehouse("demo", "live", time.Time{})},
			rows: []database.Warehouse{row}, wantErr: "incomplete identity",
		},
		{
			name: "same name in different Projects", rows: []database.Warehouse{row},
			objects: []client.Object{parent, warehouse, testWarehouse("other", "other-warehouse", created),
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
				Reader: builder.Build(), warehouseErr: testCase.warehouseErr, parentErr: testCase.parentErr,
				incompleteWarehouse: testCase.incompleteWarehouse, incompleteParent: testCase.incompleteParent,
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
	warehouseErr        error
	parentErr           error
	incompleteWarehouse bool
	incompleteParent    bool
	insertInto          *fakeStore
	calls               int
}

func (r *diffReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	r.calls++
	_, isWarehouse := list.(*kargoapi.WarehouseList)
	if isWarehouse && r.warehouseErr != nil {
		return r.warehouseErr
	}
	if !isWarehouse && r.parentErr != nil {
		return r.parentErr
	}
	if r.insertInto != nil && r.calls == 1 {
		r.insertInto.rows = append(r.insertInto.rows, database.Warehouse{ID: "concurrent", Name: "concurrent"})
	}
	if err := r.Reader.List(ctx, list, opts...); err != nil {
		return err
	}
	if (isWarehouse && r.incompleteWarehouse) || (!isWarehouse && r.incompleteParent) {
		list.SetContinue("next-page")
	}
	return nil
}
