package freight

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

func TestSync(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	project := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "demo", UID: "project-id"}}
	warehouse := &kargoapi.Warehouse{ObjectMeta: metav1.ObjectMeta{
		Namespace: "demo", Name: "images", UID: "warehouse-id",
	}}
	freight := testFreight("demo", "opaque-id", created)
	terminating := freight.DeepCopy()
	terminating.DeletionTimestamp = &metav1.Time{Time: created.Add(time.Hour)}
	testCases := []struct {
		name              string
		obj               client.Object
		project           *kargoapi.Project
		warehouse         *kargoapi.Warehouse
		storedWarehouseID string
		lookupErr         error
		writeErr          error
		readErr           error
		wantErr           string
	}{
		{name: "new Freight resolves parents", obj: freight, project: project, warehouse: warehouse},
		{name: "terminating Freight stays present", obj: terminating, project: project, warehouse: warehouse},
		{
			name: "existing association survives Warehouse deletion", obj: freight, project: project,
			storedWarehouseID: "original-warehouse",
		},
		{
			name: "existing association survives Warehouse recreation", obj: freight, project: project,
			warehouse: warehouse, storedWarehouseID: "original-warehouse",
		},
		{name: "missing Project", obj: freight, wantErr: "reading freight's project"},
		{name: "missing Warehouse", obj: freight, project: project, wantErr: "reading freight's warehouse"},
		{name: "wrong object", obj: warehouse, wantErr: "expected Freight"},
		{
			name: "Kubernetes read error", obj: freight, readErr: errors.New("offline"),
			wantErr: "reading freight's project",
		},
		{
			name: "association read error", obj: freight, project: project, lookupErr: errors.New("offline"),
			wantErr: "reading freight's stored warehouse identity",
		},
		{
			name: "parent database row not yet present", obj: freight, project: project, warehouse: warehouse,
			writeErr: errors.New("foreign key violation"), wantErr: "syncing freight",
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
			if testCase.warehouse != nil {
				builder.WithObjects(testCase.warehouse.DeepCopy())
			}
			store := &fakeStore{
				warehouseID: testCase.storedWarehouseID, lookupErr: testCase.lookupErr, writeErr: testCase.writeErr,
			}
			s := NewSyncer(&failingReader{Reader: builder.Build(), err: testCase.readErr}, store)
			require.IsType(t, &kargoapi.Freight{}, s.NewObject())
			for range 2 {
				err := s.Sync(context.Background(), testCase.obj)
				if testCase.wantErr != "" {
					require.ErrorContains(t, err, testCase.wantErr)
				} else {
					require.NoError(t, err)
				}
			}
			if testCase.wantErr == "" {
				require.Len(t, store.upserts, 2)
				require.Equal(t, store.upserts[0], store.upserts[1])
				row := store.upserts[0]
				require.Equal(t, "opaque-id", row.ID)
				require.Equal(t, "project-id", row.ProjectID)
				wantWarehouse := "warehouse-id"
				if testCase.storedWarehouseID != "" {
					wantWarehouse = testCase.storedWarehouseID
				}
				require.Equal(t, wantWarehouse, row.WarehouseID)
			} else if testCase.writeErr == nil {
				require.Empty(t, store.upserts)
			}
			require.Empty(t, store.deletes)
			require.Empty(t, store.pruned)
		})
	}
}

func TestToUpsertParams(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	project := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{UID: "project"}}
	testCases := []struct {
		name    string
		change  func(*kargoapi.Freight, *kargoapi.Project, *string)
		wantErr string
	}{
		{name: "all artifact types and opaque metadata"},
		{
			name: "explicit discovery time", change: func(f *kargoapi.Freight, _ *kargoapi.Project, _ *string) {
				f.DiscoveredAt = &metav1.Time{Time: created.Add(-time.Hour)}
			},
		},
		{
			name: "missing Freight UID", wantErr: "incomplete identity",
			change: func(f *kargoapi.Freight, _ *kargoapi.Project, _ *string) { f.UID = "" },
		},
		{
			name: "missing Project UID", wantErr: "incomplete identity",
			change: func(_ *kargoapi.Freight, p *kargoapi.Project, _ *string) { p.UID = "" },
		},
		{
			name: "missing Warehouse UID", wantErr: "incomplete identity",
			change: func(_ *kargoapi.Freight, _ *kargoapi.Project, id *string) { *id = "" },
		},
		{
			name: "missing creation time", wantErr: "incomplete identity",
			change: func(f *kargoapi.Freight, _ *kargoapi.Project, _ *string) { f.CreationTimestamp = metav1.Time{} },
		},
		{
			name: "unsupported origin", wantErr: "invalid freight origin",
			change: func(f *kargoapi.Freight, _ *kargoapi.Project, _ *string) { f.Origin.Kind = "Other" },
		},
		{
			name: "missing origin name", wantErr: "invalid freight origin",
			change: func(f *kargoapi.Freight, _ *kargoapi.Project, _ *string) { f.Origin.Name = "" },
		},
		{
			name: "invalid discovery time", wantErr: "no discovery timestamp",
			change: func(f *kargoapi.Freight, _ *kargoapi.Project, _ *string) { f.DiscoveredAt = &metav1.Time{} },
		},
		{
			name: "invalid artifact metadata", wantErr: "encoding artifact metadata",
			change: func(f *kargoapi.Freight, _ *kargoapi.Project, _ *string) {
				f.Artifacts[0].Metadata.Raw = []byte("invalid JSON")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			freight := testFreight("demo", "freight", created)
			freight.Commits = []kargoapi.GitCommit{{RepoURL: "https://example.com/repo", ID: "abc"}}
			freight.Images = []kargoapi.Image{{RepoURL: "example.com/image", Tag: "v1", Digest: "sha256:abc"}}
			freight.Charts = []kargoapi.Chart{{RepoURL: "https://example.com/charts", Name: "app", Version: "1.0"}}
			freight.Artifacts = []kargoapi.ArtifactReference{{
				ArtifactType: "custom", SubscriptionName: "custom", Version: "v1",
				Metadata: &apiextensionsv1.JSON{Raw: []byte(`{"number":9007199254740993}`)},
			}}
			parent := project.DeepCopy()
			warehouseID := "warehouse"
			if testCase.change != nil {
				testCase.change(freight, parent, &warehouseID)
			}
			params, err := toUpsertParams(freight, parent, warehouseID)
			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, freight.Name, params.Name)
			require.Equal(t, freight.Alias, params.Alias)
			require.Equal(t, created, params.CreatedAt)
			require.Equal(t, freight.EffectiveDiscoveredAt(), params.DiscoveredAt)
			require.Equal(t, []database.UpsertFreightCommitParams{{
				FreightID: "freight", RepoURL: freight.Commits[0].RepoURL, CommitID: "abc",
			}}, params.Commits)
			require.Equal(t, []database.UpsertFreightImageParams{{
				FreightID: "freight", RepoURL: "example.com/image", Tag: "v1", Digest: "sha256:abc",
				Annotations: []byte("{}"),
			}}, params.Images)
			require.Equal(t, []database.UpsertFreightChartParams{{
				FreightID: "freight", RepoURL: "https://example.com/charts", Name: "app", Version: "1.0",
			}}, params.Charts)
			require.Equal(t, []database.UpsertFreightArtifactParams{{
				FreightID: "freight", ArtifactType: "custom", SubscriptionName: "custom", Version: "v1",
				Metadata: []byte(`{"number":9007199254740993}`),
			}}, params.Artifacts)
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "storage failure"}[fail], func(t *testing.T) {
			t.Parallel()
			store := &fakeStore{}
			if fail {
				store.writeErr = errors.New("offline")
			}
			s := NewSyncer(nil, store)
			err := s.Delete(context.Background(), client.ObjectKey{Namespace: "demo", Name: "bundle"})
			require.ErrorIs(t, err, store.writeErr)
			require.Equal(t, []database.DeleteFreightByNameParams{{ProjectName: "demo", Name: "bundle"}}, store.deletes)
			err = s.DeleteByIDs(context.Background(), []string{"old-uid"})
			require.ErrorIs(t, err, store.writeErr)
			require.Equal(t, []string{"old-uid"}, store.pruned)
		})
	}
}

func testFreight(namespace, uid string, created time.Time) *kargoapi.Freight {
	return &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace, Name: "bundle", UID: types.UID(uid), CreationTimestamp: metav1.NewTime(created),
		},
		Origin: kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "images"},
		Alias:  "friendly",
	}
}

type fakeStore struct {
	rows        []database.FreightSnapshot
	upserts     []database.FreightUpsert
	deletes     []database.DeleteFreightByNameParams
	pruned      []string
	warehouseID string
	lookupErr   error
	listErr     error
	writeErr    error
}

func (s *fakeStore) UpsertFreight(_ context.Context, params database.FreightUpsert) error {
	s.upserts = append(s.upserts, params)
	return s.writeErr
}

func (s *fakeStore) GetFreightWarehouseID(context.Context, string) (string, error) {
	if s.lookupErr != nil {
		return "", s.lookupErr
	}
	if s.warehouseID == "" {
		return "", pgx.ErrNoRows
	}
	return s.warehouseID, nil
}

func (s *fakeStore) DeleteFreightByName(_ context.Context, params database.DeleteFreightByNameParams) error {
	s.deletes = append(s.deletes, params)
	return s.writeErr
}

func (s *fakeStore) ListFreight(context.Context) ([]database.FreightSnapshot, error) {
	return s.rows, s.listErr
}

func (s *fakeStore) DeleteFreightByIDs(_ context.Context, ids []string) error {
	s.pruned = append(s.pruned, ids...)
	return s.writeErr
}

type failingReader struct {
	client.Reader
	err error
}

func (r *failingReader) Get(
	ctx context.Context,
	key client.ObjectKey,
	obj client.Object,
	opts ...client.GetOption,
) error {
	if r.err != nil {
		return r.err
	}
	return r.Reader.Get(ctx, key, obj, opts...)
}
