package promotionrollouts

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/promotionrolloutstore"
	"github.com/akuity/kargo/pkg/reconciler"
	"github.com/akuity/kargo/pkg/reconciler/list"
)

// fakeStore holds rollouts in memory with the real store's precondition
// semantics: an update must carry the row's current ID and UpdatedAt, and
// Targets are resolved exactly once.
type fakeStore struct {
	mu         sync.Mutex
	rows       map[promotionrolloutstore.Key]database.PromotionRolloutRow
	targets    map[promotionrolloutstore.Key][]uuid.UUID
	getErr     error
	updateErr  error
	resolveErr error
	// updated records every row written, in order.
	updated []database.PromotionRolloutRow
	// resolved records every rollout whose Targets were resolved, in order.
	resolved []promotionrolloutstore.Key
}

var _ Store = (*fakeStore)(nil)

func newFakeStore(rows ...database.PromotionRolloutRow) *fakeStore {
	s := &fakeStore{
		rows:    map[promotionrolloutstore.Key]database.PromotionRolloutRow{},
		targets: map[promotionrolloutstore.Key][]uuid.UUID{},
	}
	for _, row := range rows {
		s.rows[promotionrolloutstore.Key{Project: row.ProjectName, Name: row.Name}] = row
	}
	return s
}

func (s *fakeStore) Get(_ context.Context, project, name string) (promotionrolloutstore.Rollout, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.getErr != nil {
		return promotionrolloutstore.Rollout{}, s.getErr
	}
	key := promotionrolloutstore.Key{Project: project, Name: name}
	row, ok := s.rows[key]
	if !ok {
		return promotionrolloutstore.Rollout{}, database.ErrNotFound
	}
	rollout := promotionrolloutstore.Rollout{
		PromotionRolloutRow: row,
		Targets:             []database.ListPromotionRolloutTargetsRow{},
	}
	for _, id := range s.targets[key] {
		rollout.Targets = append(rollout.Targets, database.ListPromotionRolloutTargetsRow{
			PromotionRolloutID: row.ID,
			TargetID:           id,
			TargetName:         id.String(),
		})
	}
	return rollout, nil
}

func (s *fakeStore) ResolveTargets(
	_ context.Context,
	rollout promotionrolloutstore.Rollout,
	ids []uuid.UUID,
) (promotionrolloutstore.Rollout, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.resolveErr != nil {
		return promotionrolloutstore.Rollout{}, s.resolveErr
	}
	key := promotionrolloutstore.Key{Project: rollout.ProjectName, Name: rollout.Name}
	current, ok := s.rows[key]
	if !ok {
		return promotionrolloutstore.Rollout{}, database.ErrNotFound
	}
	if current.TargetsResolvedAt != nil {
		return promotionrolloutstore.Rollout{}, database.ErrConflict
	}
	now := time.Now()
	current.TargetsResolvedAt = &now
	current.UpdatedAt = current.UpdatedAt.Add(time.Microsecond)
	s.rows[key] = current
	s.targets[key] = ids
	s.resolved = append(s.resolved, key)
	resolved := promotionrolloutstore.Rollout{
		PromotionRolloutRow: current,
		Targets:             []database.ListPromotionRolloutTargetsRow{},
	}
	for _, id := range ids {
		resolved.Targets = append(resolved.Targets, database.ListPromotionRolloutTargetsRow{
			PromotionRolloutID: current.ID,
			TargetID:           id,
			TargetName:         id.String(),
		})
	}
	return resolved, nil
}

func (s *fakeStore) ListOpen(context.Context) ([]promotionrolloutstore.Key, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var keys []promotionrolloutstore.Key
	for key, row := range s.rows {
		if !kargoapi.PromotionRolloutPhase(row.StatusPhase).IsTerminal() {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (s *fakeStore) UpdateStatus(
	_ context.Context,
	project string,
	row database.PromotionRolloutRow,
) (database.PromotionRolloutRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.updateErr != nil {
		return database.PromotionRolloutRow{}, s.updateErr
	}
	key := promotionrolloutstore.Key{Project: project, Name: row.Name}
	current, ok := s.rows[key]
	if !ok {
		return database.PromotionRolloutRow{}, database.ErrNotFound
	}
	if row.ID != current.ID || !row.UpdatedAt.Equal(current.UpdatedAt) {
		return database.PromotionRolloutRow{}, database.ErrConflict
	}
	current.StatusPhase = row.StatusPhase
	current.StatusReason = row.StatusReason
	current.StatusMessage = row.StatusMessage
	current.StatusStartedAt = row.StatusStartedAt
	current.StatusFinishedAt = row.StatusFinishedAt
	current.UpdatedAt = current.UpdatedAt.Add(time.Microsecond)
	s.rows[key] = current
	s.updated = append(s.updated, current)
	return current, nil
}

func (s *fakeStore) writes() []database.PromotionRolloutRow {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]database.PromotionRolloutRow(nil), s.updated...)
}

// targetsOf returns the ids the named rollout was resolved to.
func (s *fakeStore) targetsOf(key promotionrolloutstore.Key) []uuid.UUID {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.targets[key]
}

// fakeTargetLister serves a fixed set of Targets for every Project.
type fakeTargetLister struct {
	rows []database.TargetRow
	err  error
}

func (f fakeTargetLister) List(context.Context, string) ([]database.TargetRow, error) {
	return f.rows, f.err
}

var (
	usEastID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	euWestID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
)

// testTargets are the Project's Targets: one in region us, one in region eu.
var testTargets = []database.TargetRow{
	{ID: usEastID, ProjectName: "demo", Name: "us-east", Labels: map[string]string{"region": "us"}},
	{ID: euWestID, ProjectName: "demo", Name: "eu-west", Labels: map[string]string{"region": "eu"}},
}

// testStage governs the Targets in region us.
func testStage() *kargoapi.Stage {
	return &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: "demo", Name: "prod"},
		Spec: kargoapi.StageSpec{
			Targets: &kargoapi.StageTargets{
				Selectors: []metav1.LabelSelector{{MatchLabels: map[string]string{"region": "us"}}},
			},
		},
	}
}

// newKube returns a fake Kubernetes client holding the given objects.
func newKube(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

func testRow(name string, phase kargoapi.PromotionRolloutPhase) database.PromotionRolloutRow {
	return database.PromotionRolloutRow{
		ID:          uuid.New(),
		ProjectName: "demo",
		Name:        name,
		StageName:   "prod",
		Number:      1,
		FreightName: "freight",
		UpdatedAt:   time.UnixMicro(1_700_000_000_000_000),
		StatusPhase: string(phase),
	}
}

// resolvedRow is testRow with its Targets already resolved.
func resolvedRow(name string, phase kargoapi.PromotionRolloutPhase) database.PromotionRolloutRow {
	row := testRow(name, phase)
	row.TargetsResolvedAt = &row.UpdatedAt
	return row
}

func TestMarkEnterpriseOnly(t *testing.T) {
	t.Parallel()
	now := time.Now()
	started := now.Add(-time.Minute)
	row := database.PromotionRolloutRow{StatusPhase: "Running", StatusStartedAt: &started}
	markEnterpriseOnly(&row, now)
	require.Equal(t, "Errored", row.StatusPhase)
	require.Equal(t, enterpriseOnlyReason, row.StatusReason)
	require.Equal(t, enterpriseOnlyMessage, row.StatusMessage)
	require.Equal(t, &started, row.StatusStartedAt)
	require.Equal(t, &now, row.StatusFinishedAt)

	// A finish time already recorded is kept.
	markEnterpriseOnly(&row, now.Add(time.Hour))
	require.Equal(t, &now, row.StatusFinishedAt)
}

func TestReconcile(t *testing.T) {
	t.Parallel()
	key := promotionrolloutstore.Key{Project: "demo", Name: "prod.01A.abc"}
	testCases := []struct {
		name string
		// objects are the Stage, if any, the fake Kubernetes client holds.
		objects []client.Object
		targets fakeTargetLister
		store   func() *fakeStore
		assert  func(*testing.T, *fakeStore, reconciler.Result, error)
	}{
		{
			name:  "a rollout that no longer exists is done",
			store: func() *fakeStore { return newFakeStore() },
			assert: func(t *testing.T, s *fakeStore, _ reconciler.Result, err error) {
				require.NoError(t, err)
				require.Empty(t, s.writes())
			},
		},
		{
			name:    "an open rollout has its Targets resolved, then is ended as enterprise-only",
			objects: []client.Object{testStage()},
			targets: fakeTargetLister{rows: testTargets},
			store: func() *fakeStore {
				return newFakeStore(testRow(key.Name, kargoapi.PromotionRolloutPhasePending))
			},
			assert: func(t *testing.T, s *fakeStore, _ reconciler.Result, err error) {
				require.NoError(t, err)
				require.Equal(t, []promotionrolloutstore.Key{key}, s.resolved)
				require.Equal(t, []uuid.UUID{usEastID}, s.targetsOf(key))
				writes := s.writes()
				require.Len(t, writes, 1)
				require.Equal(t, "Errored", writes[0].StatusPhase)
				require.Equal(t, enterpriseOnlyReason, writes[0].StatusReason)
				require.Equal(t, enterpriseOnlyMessage, writes[0].StatusMessage)
				require.NotNil(t, writes[0].StatusFinishedAt)
				require.Nil(t, writes[0].StatusStartedAt)
				require.NotNil(t, writes[0].TargetsResolvedAt)
				require.Equal(t, "prod", writes[0].StageName)
			},
		},
		{
			name:    "a rollout already resolved is not resolved again",
			objects: []client.Object{testStage()},
			targets: fakeTargetLister{rows: testTargets},
			store: func() *fakeStore {
				return newFakeStore(resolvedRow(key.Name, kargoapi.PromotionRolloutPhasePending))
			},
			assert: func(t *testing.T, s *fakeStore, _ reconciler.Result, err error) {
				require.NoError(t, err)
				require.Empty(t, s.resolved)
				require.Len(t, s.writes(), 1)
			},
		},
		{
			name:    "a rollout whose Stage is gone resolves to no Targets",
			targets: fakeTargetLister{rows: testTargets},
			store: func() *fakeStore {
				return newFakeStore(testRow(key.Name, kargoapi.PromotionRolloutPhasePending))
			},
			assert: func(t *testing.T, s *fakeStore, _ reconciler.Result, err error) {
				require.NoError(t, err)
				require.Equal(t, []promotionrolloutstore.Key{key}, s.resolved)
				require.Empty(t, s.targetsOf(key))
			},
		},
		{
			name: "a rollout whose Stage no longer selects Targets resolves to none",
			objects: []client.Object{func() *kargoapi.Stage {
				stage := testStage()
				stage.Spec.Targets = nil
				return stage
			}()},
			targets: fakeTargetLister{rows: testTargets},
			store: func() *fakeStore {
				return newFakeStore(testRow(key.Name, kargoapi.PromotionRolloutPhasePending))
			},
			assert: func(t *testing.T, s *fakeStore, _ reconciler.Result, err error) {
				require.NoError(t, err)
				require.Empty(t, s.targetsOf(key))
			},
		},
		{
			name: "a rollout whose Stage has invalid selectors resolves to none",
			objects: []client.Object{func() *kargoapi.Stage {
				stage := testStage()
				stage.Spec.Targets.Selectors = []metav1.LabelSelector{{
					MatchExpressions: []metav1.LabelSelectorRequirement{{Key: "region", Operator: "Bogus"}},
				}}
				return stage
			}()},
			targets: fakeTargetLister{rows: testTargets},
			store: func() *fakeStore {
				return newFakeStore(testRow(key.Name, kargoapi.PromotionRolloutPhasePending))
			},
			assert: func(t *testing.T, s *fakeStore, _ reconciler.Result, err error) {
				require.NoError(t, err)
				require.Equal(t, []promotionrolloutstore.Key{key}, s.resolved)
				require.Empty(t, s.targetsOf(key))
			},
		},
		{
			name:    "a failed Target list is retried before anything is written",
			objects: []client.Object{testStage()},
			targets: fakeTargetLister{err: errors.New("database is down")},
			store: func() *fakeStore {
				return newFakeStore(testRow(key.Name, kargoapi.PromotionRolloutPhasePending))
			},
			assert: func(t *testing.T, s *fakeStore, _ reconciler.Result, err error) {
				require.ErrorContains(t, err, "database is down")
				require.Empty(t, s.resolved)
				require.Empty(t, s.writes())
			},
		},
		{
			name:    "a resolution lost to another reconciler goes with what it wrote",
			objects: []client.Object{testStage()},
			targets: fakeTargetLister{rows: testTargets},
			store: func() *fakeStore {
				// The store says the rollout is resolved, but the row this
				// reconcile read was not yet: the other reconciler got in
				// between.
				s := newFakeStore(resolvedRow(key.Name, kargoapi.PromotionRolloutPhasePending))
				s.resolveErr = database.ErrConflict
				return s
			},
			assert: func(t *testing.T, s *fakeStore, _ reconciler.Result, err error) {
				require.NoError(t, err)
				require.Len(t, s.writes(), 1)
			},
		},
		{
			name: "a rollout that has ended is left alone",
			store: func() *fakeStore {
				return newFakeStore(resolvedRow(key.Name, kargoapi.PromotionRolloutPhaseSucceeded))
			},
			assert: func(t *testing.T, s *fakeStore, _ reconciler.Result, err error) {
				require.NoError(t, err)
				require.Empty(t, s.writes())
			},
		},
		{
			name: "a second reconcile writes nothing more",
			store: func() *fakeStore {
				s := newFakeStore(resolvedRow(key.Name, kargoapi.PromotionRolloutPhaseRunning))
				_, err := newReconciler(newKube(t), s, fakeTargetLister{})(context.Background(), key)
				require.NoError(t, err)
				return s
			},
			assert: func(t *testing.T, s *fakeStore, _ reconciler.Result, err error) {
				require.NoError(t, err)
				require.Len(t, s.writes(), 1)
			},
		},
		{
			name: "a failed read is retried",
			store: func() *fakeStore {
				s := newFakeStore()
				s.getErr = errors.New("connection reset")
				return s
			},
			assert: func(t *testing.T, _ *fakeStore, _ reconciler.Result, err error) {
				require.ErrorContains(t, err, "connection reset")
			},
		},
		{
			name: "a conflicting write is retried",
			store: func() *fakeStore {
				s := newFakeStore(resolvedRow(key.Name, kargoapi.PromotionRolloutPhasePending))
				s.updateErr = database.ErrConflict
				return s
			},
			assert: func(t *testing.T, _ *fakeStore, _ reconciler.Result, err error) {
				require.ErrorIs(t, err, database.ErrConflict)
			},
		},
		{
			name: "a rollout deleted under the write is done",
			store: func() *fakeStore {
				s := newFakeStore(resolvedRow(key.Name, kargoapi.PromotionRolloutPhasePending))
				s.updateErr = database.ErrNotFound
				return s
			},
			assert: func(t *testing.T, _ *fakeStore, _ reconciler.Result, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			s := testCase.store()
			kube := newKube(t, testCase.objects...)
			result, err := newReconciler(kube, s, testCase.targets)(context.Background(), key)
			testCase.assert(t, s, result, err)
		})
	}
}

// TestControllerResync runs the reconciler on the engine with the store's
// open-rollout list as its only source, as a resync does, and checks that
// every open rollout is ended and ended ones are left alone.
func TestControllerResync(t *testing.T) {
	t.Parallel()
	s := newFakeStore(
		testRow("prod.01A.abc", kargoapi.PromotionRolloutPhasePending),
		resolvedRow("prod.01B.abc", kargoapi.PromotionRolloutPhaseRunning),
		resolvedRow("prod.01C.abc", kargoapi.PromotionRolloutPhaseSucceeded),
	)
	c, err := reconciler.New[promotionrolloutstore.Key](controllerName).
		Watch(list.New(s.ListOpen)).
		Func(newReconciler(newKube(t, testStage()), s, fakeTargetLister{rows: testTargets}))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- c.Start(ctx) }()
	require.Eventually(t, func() bool {
		return len(s.writes()) == 2
	}, 5*time.Second, 10*time.Millisecond)
	cancel()
	select {
	case err = <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("controller did not stop")
	}
	keys, err := s.ListOpen(context.Background())
	require.NoError(t, err)
	require.Empty(t, keys)
	succeeded, err := s.Get(context.Background(), "demo", "prod.01C.abc")
	require.NoError(t, err)
	require.Equal(t, "Succeeded", succeeded.StatusPhase)
	require.Equal(t, []promotionrolloutstore.Key{{Project: "demo", Name: "prod.01A.abc"}}, s.resolved)
}

func TestReconcilerConfigFromEnv(t *testing.T) {
	cfg := ReconcilerConfigFromEnv()
	require.Equal(t, time.Minute, cfg.ResyncInterval)
	require.Equal(t, 4, cfg.MaxConcurrentReconciles)

	t.Setenv("PROMOTION_ROLLOUT_RESYNC_INTERVAL", "30s")
	t.Setenv("MAX_CONCURRENT_PROMOTION_ROLLOUT_RECONCILES", "2")
	cfg = ReconcilerConfigFromEnv()
	require.Equal(t, 30*time.Second, cfg.ResyncInterval)
	require.Equal(t, 2, cfg.MaxConcurrentReconciles)
}
