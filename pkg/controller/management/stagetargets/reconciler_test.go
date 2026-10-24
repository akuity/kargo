package stagetargets

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database"
)

// fakeStore holds one Project's Stages, Targets and governance rows.
type fakeStore struct {
	stages  []database.ListStageSelectorsRow
	targets []database.ListTargetLabelsRow
	rows    []database.StageTargetRow
	// errs fails the named call.
	errs     map[string]error
	inserts  int
	deletes  int
	projects []string
}

func (s *fakeStore) fail(call string) error { return s.errs[call] }

func (s *fakeStore) ListStageSelectors(_ context.Context, project string) ([]database.ListStageSelectorsRow, error) {
	s.projects = append(s.projects, project)
	return s.stages, s.fail("stages")
}

func (s *fakeStore) ListTargetLabels(context.Context, string) ([]database.ListTargetLabelsRow, error) {
	return s.targets, s.fail("targets")
}

func (s *fakeStore) ListStageTargets(context.Context, string) ([]database.StageTargetRow, error) {
	return s.rows, s.fail("rows")
}

func (s *fakeStore) InsertStageTargets(_ context.Context, params database.InsertStageTargetsParams) error {
	if err := s.fail("insert"); err != nil {
		return err
	}
	s.inserts++
	for i := range params.StageIds {
		s.rows = append(s.rows, database.StageTargetRow{StageID: params.StageIds[i], TargetID: params.TargetIds[i]})
	}
	return nil
}

func (s *fakeStore) DeleteStageTargets(_ context.Context, params database.DeleteStageTargetsParams) error {
	if err := s.fail("delete"); err != nil {
		return err
	}
	s.deletes++
	s.rows = slices.DeleteFunc(s.rows, func(row database.StageTargetRow) bool {
		for i := range params.StageIds {
			if row.StageID == params.StageIds[i] && row.TargetID == params.TargetIds[i] {
				return true
			}
		}
		return false
	})
	return nil
}

func TestReconcile(t *testing.T) {
	t.Parallel()
	us := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	eu := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	newStore := func() *fakeStore {
		return &fakeStore{
			stages: []database.ListStageSelectorsRow{{
				ID: "fleet", Name: "fleet", TargetSelectors: json.RawMessage(`[{"matchLabels":{"region":"us"}}]`),
			}},
			targets: []database.ListTargetLabelsRow{
				{ID: us, Name: "us", Labels: map[string]string{"region": "us"}},
				{ID: eu, Name: "eu", Labels: map[string]string{"region": "eu"}},
			},
		}
	}
	governs := func(stage string, target uuid.UUID) database.StageTargetRow {
		return database.StageTargetRow{StageID: stage, TargetID: target}
	}
	testCases := []struct {
		name    string
		store   func() *fakeStore
		wantErr string
		assert  func(*testing.T, *fakeStore)
	}{
		{
			name:  "records what the selectors govern",
			store: newStore,
			assert: func(t *testing.T, s *fakeStore) {
				require.Equal(t, []database.StageTargetRow{governs("fleet", us)}, s.rows)
				require.Equal(t, 1, s.inserts)
				require.Zero(t, s.deletes)
				require.Equal(t, []string{"demo"}, s.projects)
			},
		},
		{
			name: "removes what they no longer govern and keeps the rest without writing it",
			store: func() *fakeStore {
				s := newStore()
				s.rows = []database.StageTargetRow{governs("fleet", us), governs("fleet", eu), governs("gone", us)}
				return s
			},
			assert: func(t *testing.T, s *fakeStore) {
				require.Equal(t, []database.StageTargetRow{governs("fleet", us)}, s.rows)
				require.Zero(t, s.inserts)
				require.Equal(t, 1, s.deletes)
			},
		},
		{
			name: "writes nothing when nothing changed",
			store: func() *fakeStore {
				s := newStore()
				s.rows = []database.StageTargetRow{governs("fleet", us)}
				return s
			},
			assert: func(t *testing.T, s *fakeStore) {
				require.Zero(t, s.inserts)
				require.Zero(t, s.deletes)
			},
		},
		{
			name: "a failure to list Stages fails the reconcile",
			store: func() *fakeStore {
				s := newStore()
				s.errs = map[string]error{"stages": errors.New("offline")}
				return s
			},
			wantErr: "error listing stage selectors",
		},
		{
			name: "a failure to list Targets fails the reconcile",
			store: func() *fakeStore {
				s := newStore()
				s.errs = map[string]error{"targets": errors.New("offline")}
				return s
			},
			wantErr: "error listing target labels",
		},
		{
			name: "a failure to list rows fails the reconcile",
			store: func() *fakeStore {
				s := newStore()
				s.errs = map[string]error{"rows": errors.New("offline")}
				return s
			},
			wantErr: "error listing stage targets",
		},
		{
			name: "a failure to delete fails the reconcile",
			store: func() *fakeStore {
				s := newStore()
				s.rows = []database.StageTargetRow{governs("fleet", eu)}
				s.errs = map[string]error{"delete": errors.New("offline")}
				return s
			},
			wantErr: "error deleting stage targets",
		},
		{
			name: "a failure to insert fails the reconcile",
			store: func() *fakeStore {
				s := newStore()
				s.errs = map[string]error{"insert": errors.New("offline")}
				return s
			},
			wantErr: "error inserting stage targets",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			s := testCase.store()
			r := &stageTargetsReconciler{store: s}
			result, err := r.reconcile(context.Background(), "demo")
			require.Zero(t, result)
			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
				return
			}
			require.NoError(t, err)
			testCase.assert(t, s)
		})
	}
}

func TestBatches(t *testing.T) {
	t.Parallel()
	pairs := make([]pair, 2*batchSize+1)
	var sizes []int
	for batch := range batches(pairs) {
		sizes = append(sizes, len(batch))
	}
	require.Equal(t, []int{batchSize, batchSize, 1}, sizes)
	for range batches(nil) {
		t.Fatal("no pairs, no batches")
	}
	// Stopping early stops.
	for range batches(pairs) {
		break
	}
}

func TestColumns(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	stageIDs, targetIDs := columns([]pair{{"a", id}, {"b", id}})
	require.Equal(t, []string{"a", "b"}, stageIDs)
	require.Equal(t, []uuid.UUID{id, id}, targetIDs)
}
