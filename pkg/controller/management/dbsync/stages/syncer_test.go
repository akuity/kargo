package stages

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
	created := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	terminating := testStage("opaque-id", created)
	terminating.DeletionTimestamp = &metav1.Time{Time: created.Add(time.Hour)}
	noProject := testStage("opaque-id", created)
	noProject.Namespace = ""
	replaced := database.DeleteReplacedStageParams{ProjectName: "demo", Name: "dev", ID: "opaque-id"}
	upserted := database.UpsertStageParams{ID: "opaque-id", ProjectName: "demo", Name: "dev", CreatedAt: created}
	testCases := []struct {
		name       string
		obj        client.Object
		replaceErr error
		storeErr   error
		wantErr    string
		replaces   bool
		writes     bool
	}{
		{
			name: "copies metadata without UUID conversion", obj: testStage("opaque-id", created),
			replaces: true, writes: true,
		},
		{name: "preserves terminating objects", obj: terminating, replaces: true, writes: true},
		{
			name: "failure deleting a replaced row stops the upsert", obj: testStage("opaque-id", created),
			replaceErr: errors.New("offline"), wantErr: "error deleting replaced stage", replaces: true,
		},
		{
			name: "storage failure propagates", obj: testStage("opaque-id", created),
			storeErr: errors.New("offline"), wantErr: "error syncing stage", replaces: true, writes: true,
		},
		{name: "wrong object type", obj: &kargoapi.Project{}, wantErr: "expected a Stage"},
		{name: "missing UID", obj: testStage("", created), wantErr: "incomplete identity"},
		{name: "missing namespace", obj: noProject, wantErr: "incomplete identity"},
		{name: "missing timestamp", obj: testStage("opaque-id", time.Time{}), wantErr: "incomplete identity"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			store := &fakeStore{replaceErr: testCase.replaceErr, writeErr: testCase.storeErr}
			s := NewSyncer(nil, store)
			require.IsType(t, &kargoapi.Stage{}, s.NewObject())
			require.Len(t, s.Sources(nil), 1)
			for range 2 {
				err := s.Sync(context.Background(), testCase.obj)
				if testCase.wantErr != "" {
					require.ErrorContains(t, err, testCase.wantErr)
				} else {
					require.NoError(t, err)
				}
			}
			if testCase.replaces {
				require.Equal(t, []database.DeleteReplacedStageParams{replaced, replaced}, store.replaced)
			} else {
				require.Empty(t, store.replaced)
			}
			if testCase.writes {
				require.Equal(t, []database.UpsertStageParams{upserted, upserted}, store.upserts)
				// Each upsert follows the deletion of the row it replaces.
				require.Equal(t, []string{"replace", "upsert", "replace", "upsert"}, store.calls)
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
			require.Equal(
				t,
				[]database.DeleteStageByNameParams{{ProjectName: "demo", Name: "dev"}},
				store.deletes,
			)
		})
	}
}

func testStage(uid string, created time.Time) *kargoapi.Stage {
	return &kargoapi.Stage{ObjectMeta: metav1.ObjectMeta{
		Namespace: "demo", Name: "dev", UID: types.UID(uid), CreationTimestamp: metav1.NewTime(created),
	}}
}

type fakeStore struct {
	rows       []database.StageRow
	replaced   []database.DeleteReplacedStageParams
	upserts    []database.UpsertStageParams
	deletes    []database.DeleteStageByNameParams
	calls      []string
	listErr    error
	replaceErr error
	writeErr   error
}

func (s *fakeStore) DeleteReplacedStage(_ context.Context, params database.DeleteReplacedStageParams) error {
	s.replaced = append(s.replaced, params)
	s.calls = append(s.calls, "replace")
	return s.replaceErr
}

func (s *fakeStore) UpsertStage(_ context.Context, params database.UpsertStageParams) error {
	s.upserts = append(s.upserts, params)
	s.calls = append(s.calls, "upsert")
	return s.writeErr
}

func (s *fakeStore) DeleteStageByName(_ context.Context, params database.DeleteStageByNameParams) error {
	s.deletes = append(s.deletes, params)
	return s.writeErr
}

func (s *fakeStore) ListStages(context.Context) ([]database.StageRow, error) {
	return slices.Clone(s.rows), s.listErr
}
