package freight

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
	freight := testFreight("demo", "live", created)
	parent := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "demo", UID: "parent"}}
	objects := []client.Object{parent, freight}
	row := database.FreightSnapshot{Freight: database.Freight{
		ID: "live", ProjectID: "parent", WarehouseID: "original-warehouse", Name: "bundle", Alias: "friendly",
		CreatedAt: created, DiscoveredAt: created,
	}}
	wrongAlias, wrongContents, wrongDiscovery := row, row, row
	wrongAlias.Alias = "changed"
	wrongContents.Commits = []database.UpsertFreightCommitParams{{FreightID: "live", CommitID: "other"}}
	wrongDiscovery.DiscoveredAt = created.Add(-time.Hour)
	statusChange := freight.DeepCopy()
	statusChange.Status.ApprovedFor = map[string]kargoapi.ApprovedStage{"prod": {}}
	preciseTime := freight.DeepCopy()
	preciseTime.CreationTimestamp.Time = created.Add(123 * time.Nanosecond)
	wrongName, wrongParent, wrongTimestamp, oldSyncTime := row, row, row, row
	wrongName.Name = "wrong"
	wrongParent.ProjectID = "wrong"
	wrongTimestamp.CreatedAt = created.Add(-time.Hour)
	oldSyncTime.SyncedAt = created.Add(-24 * time.Hour)
	key := client.ObjectKey{Namespace: "demo", Name: "bundle"}
	testCases := []struct {
		name              string
		objects           []client.Object
		rows              []database.FreightSnapshot
		snapshotErr       error
		freightErr        error
		parentErr         error
		incompleteFreight bool
		incompleteParent  bool
		insert            bool
		want              syncapi.Changes
		wantErr           string
	}{
		{name: "empty database", objects: objects, want: syncapi.Changes{ToSync: []client.ObjectKey{key}}},
		{name: "matching row without live Warehouse", objects: objects, rows: []database.FreightSnapshot{row}},
		{
			name: "status changes are ignored", objects: []client.Object{parent, statusChange},
			rows: []database.FreightSnapshot{row},
		},
		{
			name: "database timestamp precision", objects: []client.Object{parent, preciseTime},
			rows: []database.FreightSnapshot{row},
		},
		{name: "changed alias", objects: objects, rows: []database.FreightSnapshot{wrongAlias},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}}},
		{name: "changed contents", objects: objects, rows: []database.FreightSnapshot{wrongContents},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}}},
		{name: "changed discovery", objects: objects, rows: []database.FreightSnapshot{wrongDiscovery},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}}},
		{name: "synced_at is ignored", objects: objects, rows: []database.FreightSnapshot{oldSyncTime}},
		{
			name: "changed name", objects: objects, rows: []database.FreightSnapshot{wrongName},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}},
		},
		{
			name: "changed project UID", objects: objects, rows: []database.FreightSnapshot{wrongParent},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}},
		},
		{
			name: "changed creation timestamp", objects: objects, rows: []database.FreightSnapshot{wrongTimestamp},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}},
		},
		{
			name: "missing parent retries without deleting the Freight", objects: []client.Object{freight},
			rows: []database.FreightSnapshot{row, {Freight: database.Freight{ID: "stale"}}},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}, ToDelete: []string{"stale"}},
		},
		{
			name: "offline deletion", rows: []database.FreightSnapshot{row},
			want: syncapi.Changes{ToDelete: []string{"live"}},
		},
		{
			name: "recreation", objects: []client.Object{parent, testFreight("demo", "new", created)},
			rows: []database.FreightSnapshot{row},
			want: syncapi.Changes{ToSync: []client.ObjectKey{key}, ToDelete: []string{"live"}},
		},
		{name: "insert after snapshot", objects: objects, rows: []database.FreightSnapshot{row}, insert: true},
		{name: "database failure", snapshotErr: errors.New("offline"), wantErr: "listing mirrored freight"},
		{
			name: "Freight list failure", rows: []database.FreightSnapshot{row},
			freightErr: errors.New("offline"), wantErr: "listing objects",
		},
		{
			name: "parent list failure", rows: []database.FreightSnapshot{row},
			parentErr: errors.New("offline"), wantErr: "listing freight parents",
		},
		{
			name: "incomplete Freight list", rows: []database.FreightSnapshot{row},
			incompleteFreight: true, wantErr: "incomplete Kubernetes list",
		},
		{
			name: "incomplete parent list", rows: []database.FreightSnapshot{row},
			incompleteParent: true, wantErr: "incomplete freight parent list",
		},
		{
			name: "invalid identity", objects: []client.Object{parent, testFreight("demo", "", created)},
			rows: []database.FreightSnapshot{row}, wantErr: "has no UID",
		},
		{
			name: "invalid timestamp", objects: []client.Object{parent, testFreight("demo", "live", time.Time{})},
			rows: []database.FreightSnapshot{row}, wantErr: "incomplete identity",
		},
		{
			name: "same name in different Projects", rows: []database.FreightSnapshot{row},
			objects: []client.Object{parent, freight, testFreight("other", "other-freight", created),
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "other", UID: "other-parent"}}},
			want: syncapi.Changes{ToSync: []client.ObjectKey{{Namespace: "other", Name: "bundle"}}},
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
				Reader: builder.Build(), freightErr: testCase.freightErr, parentErr: testCase.parentErr,
				incompleteFreight: testCase.incompleteFreight, incompleteParent: testCase.incompleteParent,
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
	freightErr        error
	parentErr         error
	incompleteFreight bool
	incompleteParent  bool
	insertInto        *fakeStore
	calls             int
}

func (r *diffReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	r.calls++
	_, isFreight := list.(*kargoapi.FreightList)
	if isFreight && r.freightErr != nil {
		return r.freightErr
	}
	if !isFreight && r.parentErr != nil {
		return r.parentErr
	}
	if r.insertInto != nil && r.calls == 1 {
		r.insertInto.rows = append(r.insertInto.rows, database.FreightSnapshot{
			Freight: database.Freight{ID: "concurrent", Name: "concurrent"},
		})
	}
	if err := r.Reader.List(ctx, list, opts...); err != nil {
		return err
	}
	if (isFreight && r.incompleteFreight) || (!isFreight && r.incompleteParent) {
		list.SetContinue("next-page")
	}
	return nil
}
