package projects

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

func TestSync(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	terminating := testProject("opaque-id", created)
	terminating.DeletionTimestamp = &metav1.Time{Time: created.Add(time.Hour)}
	testCases := []struct {
		name     string
		obj      client.Object
		storeErr error
		wantErr  string
		writes   bool
	}{
		{name: "copies metadata without UUID conversion", obj: testProject("opaque-id", created), writes: true},
		{name: "preserves terminating objects", obj: terminating, writes: true},
		{
			name: "storage failure propagates", obj: testProject("opaque-id", created),
			storeErr: errors.New("offline"), wantErr: "error syncing project", writes: true,
		},
		{name: "wrong object type", obj: &kargoapi.Stage{}, wantErr: "expected a Project"},
		{name: "missing UID", obj: testProject("", created), wantErr: "incomplete identity"},
		{name: "missing timestamp", obj: testProject("opaque-id", time.Time{}), wantErr: "incomplete identity"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			store := &fakeStore{writeErr: testCase.storeErr}
			s := NewSyncer(nil, store)
			require.IsType(t, &kargoapi.Project{}, s.NewObject())
			for range 2 {
				err := s.Sync(context.Background(), testCase.obj)
				if testCase.wantErr != "" {
					require.ErrorContains(t, err, testCase.wantErr)
				} else {
					require.NoError(t, err)
				}
			}
			if testCase.writes {
				require.Equal(t, []database.UpsertProjectParams{
					{ID: "opaque-id", Name: "demo", CreatedAt: created},
					{ID: "opaque-id", Name: "demo", CreatedAt: created},
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
			err := s.Delete(context.Background(), client.ObjectKey{Name: "demo"})
			require.ErrorIs(t, err, store.writeErr)
			require.Equal(t, []string{"demo"}, store.deletes)
			err = s.DeleteByIDs(context.Background(), []string{"old-uid"})
			require.ErrorIs(t, err, store.writeErr)
			require.Equal(t, []string{"old-uid"}, store.pruned)
		})
	}
}

func testProject(uid string, created time.Time) *kargoapi.Project {
	return &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{
		Name: "demo", UID: types.UID(uid), CreationTimestamp: metav1.NewTime(created),
	}}
}

type fakeStore struct {
	rows     []database.Project
	upserts  []database.UpsertProjectParams
	deletes  []string
	pruned   []string
	listErr  error
	writeErr error
}

func (s *fakeStore) UpsertProject(_ context.Context, params database.UpsertProjectParams) error {
	s.upserts = append(s.upserts, params)
	return s.writeErr
}

func (s *fakeStore) DeleteProjectByName(_ context.Context, name string) error {
	s.deletes = append(s.deletes, name)
	return s.writeErr
}

func (s *fakeStore) ListProjects(context.Context) ([]database.Project, error) {
	return slices.Clone(s.rows), s.listErr
}

func (s *fakeStore) DeleteProjectsByID(_ context.Context, ids []string) error {
	s.pruned = append(s.pruned, ids...)
	return s.writeErr
}
