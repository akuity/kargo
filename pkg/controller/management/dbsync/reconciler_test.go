package dbsync

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/management/dbsync/internal/syncapi"
)

func TestReconcile(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		absent      bool
		terminating bool
		readErr     error
		syncErr     error
		deleteErr   error
		wantErr     string
	}{
		{name: "syncs the current object"},
		{name: "preserves terminating objects", terminating: true},
		{name: "deletes only absent objects", absent: true},
		{name: "read failure never deletes", readErr: errors.New("offline"), wantErr: "reading object"},
		{name: "sync errors propagate for retry", syncErr: errors.New("sync failed"), wantErr: "sync failed"},
		{
			name: "delete errors propagate for retry", absent: true,
			deleteErr: errors.New("delete failed"), wantErr: "delete failed",
		},
	}
	for _, isProject := range []bool{true, false} {
		kind := map[bool]string{true: "Project", false: "Stage"}[isProject]
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			for _, testCase := range testCases {
				t.Run(testCase.name, func(t *testing.T) {
					t.Parallel()
					scheme := runtime.NewScheme()
					require.NoError(t, kargoapi.AddToScheme(scheme))
					s := &fakeSyncer{project: isProject, syncErr: testCase.syncErr, deleteErr: testCase.deleteErr}
					obj := s.NewObject()
					obj.SetName("demo")
					obj.SetUID("current-uid")
					if !isProject {
						obj.SetNamespace("project")
					}
					if testCase.terminating {
						now := metav1.Now()
						obj.SetDeletionTimestamp(&now)
						obj.SetFinalizers([]string{"test.example/finalizer"})
					}
					builder := fake.NewClientBuilder().WithScheme(scheme)
					if !testCase.absent {
						builder.WithObjects(obj)
					}
					reader := &failingReader{Reader: builder.Build(), err: testCase.readErr}
					r := newReconciler(reader, s)
					key := client.ObjectKeyFromObject(obj)
					result, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: key})
					if testCase.wantErr != "" {
						require.ErrorContains(t, err, testCase.wantErr)
					} else {
						require.NoError(t, err)
					}
					require.Zero(t, result)
					switch {
					case testCase.readErr != nil:
						require.Empty(t, s.synced)
						require.Empty(t, s.deleted)
					case testCase.absent:
						require.Equal(t, []client.ObjectKey{key}, s.deleted)
						require.Empty(t, s.synced)
					default:
						require.Len(t, s.synced, 1)
						require.Equal(t, obj.GetUID(), s.synced[0].GetUID())
						require.Equal(t, key, client.ObjectKeyFromObject(s.synced[0]))
						require.Empty(t, s.deleted)
					}
				})
			}
		})
	}
}

func TestReconcileRecreation(t *testing.T) {
	t.Parallel()
	for _, isProject := range []bool{true, false} {
		t.Run(map[bool]string{true: "Project", false: "Stage"}[isProject], func(t *testing.T) {
			t.Parallel()
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))
			s := &fakeSyncer{project: isProject}
			obj := s.NewObject()
			obj.SetName("demo")
			obj.SetUID("old")
			if !isProject {
				obj.SetNamespace("project")
			}
			kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(obj).Build()
			r := newReconciler(kube, s)
			req := reconcile.Request{NamespacedName: client.ObjectKeyFromObject(obj)}
			_, err := r.Reconcile(context.Background(), req)
			require.NoError(t, err)
			require.NoError(t, kube.Delete(context.Background(), obj))
			obj.SetResourceVersion("")
			obj.SetUID("new")
			require.NoError(t, kube.Create(context.Background(), obj))
			_, err = r.Reconcile(context.Background(), req)
			require.NoError(t, err)
			require.Equal(t, "old", string(s.synced[0].GetUID()))
			require.Equal(t, "new", string(s.synced[1].GetUID()))
			require.Empty(t, s.deleted)
		})
	}
}

type fakeSyncer struct {
	project       bool
	synced        []client.Object
	deleted       []client.ObjectKey
	pruned        []string
	changes       syncapi.Changes
	syncErr       error
	deleteErr     error
	diffErr       error
	pruneErr      error
	diffCalls     int
	failFirstDiff bool
}

func (s *fakeSyncer) NewObject() client.Object {
	if s.project {
		return &kargoapi.Project{}
	}
	return &kargoapi.Stage{}
}

func (s *fakeSyncer) Sync(_ context.Context, obj client.Object) error {
	s.synced = append(s.synced, obj)
	return s.syncErr
}

func (s *fakeSyncer) Delete(_ context.Context, key client.ObjectKey) error {
	s.deleted = append(s.deleted, key)
	return s.deleteErr
}

func (s *fakeSyncer) Diff(context.Context) (syncapi.Changes, error) {
	s.diffCalls++
	if s.failFirstDiff && s.diffCalls == 1 {
		return syncapi.Changes{}, errors.New("database unavailable")
	}
	return s.changes, s.diffErr
}

func (s *fakeSyncer) DeleteByIDs(_ context.Context, ids []string) error {
	s.pruned = append(s.pruned, ids...)
	return s.pruneErr
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
