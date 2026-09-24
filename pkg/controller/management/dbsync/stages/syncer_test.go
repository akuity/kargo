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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

func TestSync(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	project := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "demo", UID: "project-id"}}
	stage := testStage("demo", "opaque-id", created)
	terminating := stage.DeepCopy()
	terminating.DeletionTimestamp = &metav1.Time{Time: created.Add(time.Hour)}
	testCases := []struct {
		name     string
		obj      client.Object
		project  *kargoapi.Project
		readErr  error
		storeErr error
		wantErr  string
		writes   bool
	}{
		{name: "copies metadata and resolves parent UID", obj: stage, project: project, writes: true},
		{name: "preserves terminating Stage", obj: terminating, project: project, writes: true},
		{name: "missing parent retries", obj: stage, wantErr: "error reading stage's project"},
		{
			name: "parent read failure", obj: stage, project: project,
			readErr: errors.New("offline"), wantErr: "error reading stage's project",
		},
		{
			name: "missing parent database row retries", obj: stage, project: project,
			storeErr: errors.New("foreign key violation"), wantErr: "error syncing stage", writes: true,
		},
		{name: "wrong object type", obj: project, wantErr: "expected a Stage"},
		{name: "missing UID", obj: testStage("demo", "", created), project: project, wantErr: "incomplete identity"},
		{
			name: "missing timestamp", obj: testStage("demo", "opaque-id", time.Time{}),
			project: project, wantErr: "incomplete identity",
		},
		{
			name: "missing parent UID", obj: stage,
			project: &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "demo"}}, wantErr: "incomplete identity",
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
			reader := &failingReader{Reader: builder.Build(), err: testCase.readErr}
			store := &fakeStore{writeErr: testCase.storeErr}
			s := NewSyncer(reader, store)
			require.IsType(t, &kargoapi.Stage{}, s.NewObject())
			for range 2 {
				err := s.Sync(context.Background(), testCase.obj)
				if testCase.wantErr != "" {
					require.ErrorContains(t, err, testCase.wantErr)
				} else {
					require.NoError(t, err)
				}
			}
			if testCase.writes {
				require.Equal(t, []database.UpsertStageParams{
					{ID: "opaque-id", ProjectID: "project-id", Name: "dev", CreatedAt: created},
					{ID: "opaque-id", ProjectID: "project-id", Name: "dev", CreatedAt: created},
				}, store.upserts)
			} else {
				require.Empty(t, store.upserts)
			}
			require.Empty(t, store.deletes)
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
			err := s.Delete(context.Background(), client.ObjectKey{Namespace: "demo", Name: "dev"})
			require.ErrorIs(t, err, store.writeErr)
			require.Equal(t, []database.DeleteStageByNameParams{{ProjectName: "demo", Name: "dev"}}, store.deletes)
			err = s.DeleteByIDs(context.Background(), []string{"old-uid"})
			require.ErrorIs(t, err, store.writeErr)
			require.Equal(t, []string{"old-uid"}, store.pruned)
		})
	}
}

func TestSyncNamespacesAndRecreation(t *testing.T) {
	t.Parallel()
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	kube := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := &fakeStore{}
	s := NewSyncer(kube, store)
	for _, namespace := range []string{"demo", "other"} {
		project := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: namespace, UID: types.UID(namespace)}}
		require.NoError(t, kube.Create(context.Background(), project))
		for _, uid := range []string{"old-" + namespace, "new-" + namespace} {
			require.NoError(t, s.Sync(context.Background(), testStage(namespace, uid, time.Now())))
			last := store.upserts[len(store.upserts)-1]
			require.Equal(t, uid, last.ID)
			require.Equal(t, namespace, last.ProjectID)
		}
	}
}

func testStage(namespace, uid string, created time.Time) *kargoapi.Stage {
	return &kargoapi.Stage{ObjectMeta: metav1.ObjectMeta{
		Namespace: namespace, Name: "dev", UID: types.UID(uid), CreationTimestamp: metav1.NewTime(created),
	}}
}

type fakeStore struct {
	rows     []database.Stage
	upserts  []database.UpsertStageParams
	deletes  []database.DeleteStageByNameParams
	pruned   []string
	listErr  error
	writeErr error
}

func (s *fakeStore) UpsertStage(_ context.Context, params database.UpsertStageParams) error {
	s.upserts = append(s.upserts, params)
	return s.writeErr
}

func (s *fakeStore) DeleteStageByName(_ context.Context, params database.DeleteStageByNameParams) error {
	s.deletes = append(s.deletes, params)
	return s.writeErr
}

func (s *fakeStore) ListStages(context.Context) ([]database.Stage, error) {
	return slices.Clone(s.rows), s.listErr
}

func (s *fakeStore) DeleteStagesByID(_ context.Context, ids []string) error {
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
