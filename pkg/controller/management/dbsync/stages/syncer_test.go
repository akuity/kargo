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
	"sigs.k8s.io/controller-runtime/pkg/event"

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

func TestTargetSelectors(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		targets *kargoapi.StageTargets
		want    string
	}{
		{name: "a classic Stage has none, stored as NULL"},
		{name: "no selectors is an empty list", targets: &kargoapi.StageTargets{}, want: `[]`},
		{
			name: "selectors are stored as written",
			targets: &kargoapi.StageTargets{Selectors: []metav1.LabelSelector{
				{MatchLabels: map[string]string{"region": "us"}},
				{MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key: "tier", Operator: metav1.LabelSelectorOpIn, Values: []string{"canary"},
				}}},
			}},
			want: `[{"matchLabels":{"region":"us"}},` +
				`{"matchExpressions":[{"key":"tier","operator":"In","values":["canary"]}]}]`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			stage := testStage("opaque-id", time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC))
			stage.Spec.Targets = testCase.targets
			params, err := toUpsertParams(stage)
			require.NoError(t, err)
			if testCase.want == "" {
				require.Nil(t, params.TargetSelectors)
				return
			}
			require.JSONEq(t, testCase.want, string(params.TargetSelectors))
		})
	}
}

func TestSpecChanged(t *testing.T) {
	t.Parallel()
	stage := func(generation int64, mutate func(*kargoapi.Stage)) *kargoapi.Stage {
		s := &kargoapi.Stage{ObjectMeta: metav1.ObjectMeta{Name: "dev", Generation: generation}}
		if mutate != nil {
			mutate(s)
		}
		return s
	}
	testCases := []struct {
		name string
		old  *kargoapi.Stage
		new  *kargoapi.Stage
		want bool
	}{
		{
			name: "a spec change passes",
			old:  stage(1, nil),
			new:  stage(2, nil),
			want: true,
		},
		{
			// The Stage's controller updates its status constantly, and
			// nothing in the row comes from it.
			name: "a status change is dropped",
			old:  stage(1, nil),
			new: stage(1, func(s *kargoapi.Stage) {
				s.Status.LastHandledRefresh = "refreshed"
			}),
		},
		{
			name: "a metadata change is dropped",
			old:  stage(1, nil),
			new: stage(1, func(s *kargoapi.Stage) {
				s.Annotations = map[string]string{"kargo.akuity.io/refresh": "now"}
			}),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.want, specChanged().Update(event.TypedUpdateEvent[*kargoapi.Stage]{
				ObjectOld: testCase.old,
				ObjectNew: testCase.new,
			}))
		})
	}
	// Creation and deletion always pass: a row is written for every Stage and
	// removed once it is gone.
	require.True(t, specChanged().Create(event.TypedCreateEvent[*kargoapi.Stage]{Object: stage(1, nil)}))
	require.True(t, specChanged().Delete(event.TypedDeleteEvent[*kargoapi.Stage]{Object: stage(1, nil)}))
}
