package promotionrequests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/dbreconcile"
)

// fakeStore is an in-memory promotionRequestStore.
type fakeStore struct {
	mu        sync.Mutex
	requests  map[uuid.UUID]database.PromotionRequestSnapshot
	getErr    error
	listErr   error
	updateErr error
	updates   int
}

func newFakeStore(snapshots ...database.PromotionRequestSnapshot) *fakeStore {
	s := &fakeStore{requests: make(map[uuid.UUID]database.PromotionRequestSnapshot)}
	for _, snapshot := range snapshots {
		s.requests[snapshot.ID] = snapshot
	}
	return s
}

func (s *fakeStore) GetPromotionRequestByID(
	_ context.Context,
	id uuid.UUID,
) (database.PromotionRequestSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.getErr != nil {
		return database.PromotionRequestSnapshot{}, s.getErr
	}
	snapshot, ok := s.requests[id]
	if !ok {
		return database.PromotionRequestSnapshot{}, fmt.Errorf("PromotionRequest %s: %w", id, database.ErrNotFound)
	}
	return snapshot, nil
}

func (s *fakeStore) ListOpenPromotionRequestIDs(
	_ context.Context,
	after uuid.UUID,
	limit int,
) ([]uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listErr != nil {
		return nil, s.listErr
	}
	var ids []uuid.UUID
	for id, snapshot := range s.requests {
		phase := kargoapi.PromotionRequestPhase(snapshot.Phase)
		if !phase.IsTerminal() && id.String() > after.String() {
			ids = append(ids, id)
		}
	}
	// Paging by id, as the database does.
	for i := range ids {
		for j := i + 1; j < len(ids); j++ {
			if ids[j].String() < ids[i].String() {
				ids[i], ids[j] = ids[j], ids[i]
			}
		}
	}
	if len(ids) > limit {
		ids = ids[:limit]
	}
	return ids, nil
}

func (s *fakeStore) UpdatePromotionRequestStatus(
	_ context.Context,
	id uuid.UUID,
	status kargoapi.PromotionRequestStatus,
) (database.PromotionRequestSnapshot, database.PromotionRequestSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.updateErr != nil {
		return database.PromotionRequestSnapshot{}, database.PromotionRequestSnapshot{}, s.updateErr
	}
	before, ok := s.requests[id]
	if !ok {
		return database.PromotionRequestSnapshot{}, database.PromotionRequestSnapshot{},
			fmt.Errorf("PromotionRequest %s: %w", id, database.ErrNotFound)
	}
	after := before
	after.Phase = string(status.Phase)
	after.Message = status.Message
	after.UpdatedAt = before.UpdatedAt.Add(time.Second)
	s.requests[id] = after
	s.updates++
	return before, after, nil
}

func (s *fakeStore) phase(id uuid.UUID) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requests[id].Phase
}

func (s *fakeStore) updateCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updates
}

type fakeHandler struct {
	status kargoapi.PromotionRequestStatus
	err    error
	seen   []*kargoapi.PromotionRequest
}

func (h *fakeHandler) Handle(
	_ context.Context,
	request *kargoapi.PromotionRequest,
) (kargoapi.PromotionRequestStatus, error) {
	h.seen = append(h.seen, request)
	return h.status, h.err
}

type fakeManager struct {
	added []manager.Runnable
}

func (m *fakeManager) Add(r manager.Runnable) error {
	m.added = append(m.added, r)
	return nil
}

func testSnapshot(phase kargoapi.PromotionRequestPhase) database.PromotionRequestSnapshot {
	return database.PromotionRequestSnapshot{
		PromotionRequest: database.PromotionRequest{
			ID:        uuid.New(),
			Name:      "dev.01j",
			Phase:     string(phase),
			CreatedAt: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
			UpdatedAt: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
		},
		ProjectName: "demo",
		Stage:       "dev",
		Freight:     "abc",
	}
}

func connect(t *testing.T) *nats.Conn {
	t.Helper()
	srv := natstest.RunRandClientPortServer()
	t.Cleanup(srv.Shutdown)
	conn, err := nats.Connect(srv.ClientURL())
	require.NoError(t, err)
	t.Cleanup(conn.Close)
	return conn
}

// nextEvent receives the next event published to a subscription.
func nextEvent(t *testing.T, sub *nats.Subscription) Event {
	t.Helper()
	msg, err := sub.NextMsg(time.Second)
	require.NoError(t, err)
	e := Event{}
	require.NoError(t, json.Unmarshal(msg.Data, &e))
	return e
}

func TestReconcilerConfigFromEnv(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		cfg := ReconcilerConfigFromEnv()
		require.Equal(t, 2*time.Minute, cfg.Interval)
		require.Equal(t, 100, cfg.ResyncPageSize)
		require.Equal(t, 4, cfg.MaxConcurrentReconciles)
	})
	t.Run("from environment", func(t *testing.T) {
		t.Setenv("PROMOTION_REQUEST_RECONCILE_INTERVAL", "30s")
		t.Setenv("PROMOTION_REQUEST_RESYNC_PAGE_SIZE", "7")
		t.Setenv("MAX_CONCURRENT_PROMOTION_REQUEST_RECONCILES", "2")
		cfg := ReconcilerConfigFromEnv()
		require.Equal(t, 30*time.Second, cfg.Interval)
		require.Equal(t, 7, cfg.ResyncPageSize)
		require.Equal(t, 2, cfg.MaxConcurrentReconciles)
	})
}

func TestEnterpriseOnlyHandler(t *testing.T) {
	t.Parallel()
	finished := metav1.NewTime(time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC))
	testCases := []struct {
		name    string
		request *kargoapi.PromotionRequest
		assert  func(*testing.T, kargoapi.PromotionRequestStatus)
	}{
		{
			name: "open request is ended",
			request: &kargoapi.PromotionRequest{
				Status: kargoapi.PromotionRequestStatus{Phase: kargoapi.PromotionRequestPhasePending},
			},
			assert: func(t *testing.T, status kargoapi.PromotionRequestStatus) {
				require.Equal(t, kargoapi.PromotionRequestPhaseErrored, status.Phase)
				require.Equal(t, enterpriseOnlyMessage, status.Message)
				require.NotNil(t, status.FinishedAt)
				require.WithinDuration(t, time.Now(), status.FinishedAt.Time, time.Minute)
				require.Nil(t, status.Conditions)
			},
		},
		{
			name: "an existing finish time is kept",
			request: &kargoapi.PromotionRequest{
				Status: kargoapi.PromotionRequestStatus{
					Phase:      kargoapi.PromotionRequestPhaseRunning,
					FinishedAt: &finished,
					Targets:    []kargoapi.PromotionRequestTargetStatus{{Name: "us-east"}},
				},
			},
			assert: func(t *testing.T, status kargoapi.PromotionRequestStatus) {
				require.Equal(t, kargoapi.PromotionRequestPhaseErrored, status.Phase)
				require.Equal(t, &finished, status.FinishedAt)
				// Everything else is left alone.
				require.Equal(t, []kargoapi.PromotionRequestTargetStatus{{Name: "us-east"}}, status.Targets)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			before := testCase.request.DeepCopy()
			status, err := NewEnterpriseOnlyHandler().Handle(context.Background(), testCase.request)
			require.NoError(t, err)
			testCase.assert(t, status)
			// The handler returns a new status; it does not write to the request.
			require.Equal(t, before, testCase.request)
		})
	}
}

func TestRequestPredicate(t *testing.T) {
	t.Parallel()
	// request builds a request with the given phase and Targets, each Target
	// given the promotion phase the reconciler has recorded for it.
	request := func(phase kargoapi.PromotionRequestPhase, targets ...string) *database.PromotionRequestSnapshot {
		snapshot := testSnapshot(phase)
		for i, name := range targets {
			snapshot.Targets = append(snapshot.Targets, database.PromotionRequestTargetRow{
				Name:    name,
				Ordinal: int64(i),
				Phase:   string(kargoapi.PromotionPhaseRunning),
			})
		}
		return &snapshot
	}
	withTargetPhase := func(
		s *database.PromotionRequestSnapshot,
		phase kargoapi.PromotionPhase,
	) *database.PromotionRequestSnapshot {
		for i := range s.Targets {
			s.Targets[i].Phase = string(phase)
		}
		return s
	}
	testCases := []struct {
		name    string
		event   Event
		allowed bool
	}{
		{
			name:    "a new request",
			event:   Event{Kind: dbreconcile.Created, New: request("", "us-east")},
			allowed: true,
		},
		{
			name: "a Target joining the request",
			event: Event{
				Kind: dbreconcile.Updated,
				Old:  request(kargoapi.PromotionRequestPhaseRunning, "us-east"),
				New:  request(kargoapi.PromotionRequestPhaseRunning, "us-east", "eu-west"),
			},
			allowed: true,
		},
		{
			// What the reconciler's own writes look like.
			name: "a change to the request's phase only",
			event: Event{
				Kind: dbreconcile.Updated,
				Old:  request(kargoapi.PromotionRequestPhasePending, "us-east"),
				New:  request(kargoapi.PromotionRequestPhaseErrored, "us-east"),
			},
		},
		{
			// Also written by the reconciler, as it records each Target's outcome.
			name: "a change to a Target's outcome only",
			event: Event{
				Kind: dbreconcile.Updated,
				Old:  request(kargoapi.PromotionRequestPhaseRunning, "us-east"),
				New: withTargetPhase(
					request(kargoapi.PromotionRequestPhaseRunning, "us-east"),
					kargoapi.PromotionPhaseSucceeded,
				),
			},
		},
		{
			name:  "a deleted request",
			event: Event{Kind: dbreconcile.Deleted, Old: request("", "us-east")},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.allowed, requestPredicate.Allow(testCase.event))
		})
	}
}

func TestPublishCreated(t *testing.T) {
	t.Parallel()
	require.NoError(t, PublishCreated(nil, testSnapshot(kargoapi.PromotionRequestPhasePending)))

	conn := connect(t)
	sub, err := conn.SubscribeSync(Subject)
	require.NoError(t, err)
	snapshot := testSnapshot(kargoapi.PromotionRequestPhasePending)
	require.NoError(t, PublishCreated(conn, snapshot))
	e := nextEvent(t, sub)
	require.Equal(t, dbreconcile.Created, e.Kind)
	require.Equal(t, snapshot.ID, e.Key)
	require.Nil(t, e.Old)
	// The event carries the request exactly as the database holds it.
	require.Equal(t, &snapshot, e.New)
}

func TestSetup(t *testing.T) {
	t.Parallel()
	conn := connect(t)
	cfg := ReconcilerConfig{Interval: time.Minute, ResyncPageSize: 10, MaxConcurrentReconciles: 1}

	require.EqualError(
		t,
		SetupWithManager(&fakeManager{}, nil, NewEnterpriseOnlyHandler(), conn, cfg),
		"PromotionRequest reconciler requires a store",
	)
	require.EqualError(
		t,
		setup(&fakeManager{}, newFakeStore(), nil, conn, cfg),
		"PromotionRequest reconciler requires a status handler",
	)
	require.EqualError(
		t,
		setup(&fakeManager{}, newFakeStore(), NewEnterpriseOnlyHandler(), nil, cfg),
		"PromotionRequest reconciler requires a NATS connection",
	)
	require.ErrorContains(
		t,
		setup(&fakeManager{}, newFakeStore(), NewEnterpriseOnlyHandler(), conn, ReconcilerConfig{}),
		"error setting up PromotionRequest reconciler",
	)

	mgr := &fakeManager{}
	require.NoError(t, setup(mgr, newFakeStore(), NewEnterpriseOnlyHandler(), conn, cfg))
	require.Len(t, mgr.added, 1)
}

func TestReconcile(t *testing.T) {
	t.Parallel()
	errored := kargoapi.PromotionRequestStatus{
		Phase:   kargoapi.PromotionRequestPhaseErrored,
		Message: enterpriseOnlyMessage,
	}
	testCases := []struct {
		name     string
		snapshot database.PromotionRequestSnapshot
		store    func(*fakeStore)
		handler  *fakeHandler
		assert   func(*testing.T, *fakeStore, *fakeHandler, *nats.Subscription, error)
	}{
		{
			name:     "request no longer exists",
			snapshot: testSnapshot(kargoapi.PromotionRequestPhasePending),
			store:    func(s *fakeStore) { s.requests = map[uuid.UUID]database.PromotionRequestSnapshot{} },
			handler:  &fakeHandler{status: errored},
			assert: func(t *testing.T, s *fakeStore, h *fakeHandler, _ *nats.Subscription, err error) {
				require.NoError(t, err)
				require.Empty(t, h.seen)
				require.Zero(t, s.updateCount())
			},
		},
		{
			name:     "error reading the request",
			snapshot: testSnapshot(kargoapi.PromotionRequestPhasePending),
			store:    func(s *fakeStore) { s.getErr = errors.New("offline") },
			handler:  &fakeHandler{status: errored},
			assert: func(t *testing.T, _ *fakeStore, h *fakeHandler, _ *nats.Subscription, err error) {
				require.ErrorContains(t, err, "error getting PromotionRequest")
				require.ErrorContains(t, err, "offline")
				require.Empty(t, h.seen)
			},
		},
		{
			name:     "a finished request is left alone",
			snapshot: testSnapshot(kargoapi.PromotionRequestPhaseSucceeded),
			handler:  &fakeHandler{status: errored},
			assert: func(t *testing.T, s *fakeStore, h *fakeHandler, _ *nats.Subscription, err error) {
				require.NoError(t, err)
				require.Empty(t, h.seen)
				require.Zero(t, s.updateCount())
			},
		},
		{
			name:     "handler failure",
			snapshot: testSnapshot(kargoapi.PromotionRequestPhasePending),
			handler:  &fakeHandler{err: errors.New("cannot decide")},
			assert: func(t *testing.T, s *fakeStore, _ *fakeHandler, _ *nats.Subscription, err error) {
				require.ErrorContains(t, err, `error handling PromotionRequest "dev.01j" in Project "demo": cannot decide`)
				require.Zero(t, s.updateCount())
			},
		},
		{
			name:     "unchanged status is not written",
			snapshot: testSnapshot(kargoapi.PromotionRequestPhasePending),
			handler:  &fakeHandler{status: kargoapi.PromotionRequestStatus{Phase: kargoapi.PromotionRequestPhasePending}},
			assert: func(t *testing.T, s *fakeStore, h *fakeHandler, sub *nats.Subscription, err error) {
				require.NoError(t, err)
				require.Zero(t, s.updateCount())
				// The handler was given the request in resource form.
				require.Len(t, h.seen, 1)
				require.Equal(t, "demo", h.seen[0].Namespace)
				require.Equal(t, "dev.01j", h.seen[0].Name)
				_, err = sub.NextMsg(20 * time.Millisecond)
				require.ErrorIs(t, err, nats.ErrTimeout)
			},
		},
		{
			name:     "changed status is written and announced",
			snapshot: testSnapshot(kargoapi.PromotionRequestPhasePending),
			handler:  &fakeHandler{status: errored},
			assert: func(t *testing.T, s *fakeStore, h *fakeHandler, sub *nats.Subscription, err error) {
				require.NoError(t, err)
				require.Equal(t, 1, s.updateCount())
				require.Equal(t, string(kargoapi.PromotionRequestPhaseErrored), s.phase(uuid.MustParse(string(h.seen[0].UID))))
				e := nextEvent(t, sub)
				require.Equal(t, dbreconcile.Updated, e.Kind)
				require.Equal(t, string(kargoapi.PromotionRequestPhasePending), e.Old.Phase)
				require.Equal(t, string(kargoapi.PromotionRequestPhaseErrored), e.New.Phase)
				require.Equal(t, enterpriseOnlyMessage, e.New.Message)
				// The announcement would not cause another reconcile.
				require.False(t, requestPredicate.Allow(e))
			},
		},
		{
			name:     "request deleted before the write",
			snapshot: testSnapshot(kargoapi.PromotionRequestPhasePending),
			store:    func(s *fakeStore) { s.updateErr = fmt.Errorf("gone: %w", database.ErrNotFound) },
			handler:  &fakeHandler{status: errored},
			assert: func(t *testing.T, _ *fakeStore, _ *fakeHandler, sub *nats.Subscription, err error) {
				require.NoError(t, err)
				_, err = sub.NextMsg(20 * time.Millisecond)
				require.ErrorIs(t, err, nats.ErrTimeout)
			},
		},
		{
			name:     "write failure",
			snapshot: testSnapshot(kargoapi.PromotionRequestPhasePending),
			store:    func(s *fakeStore) { s.updateErr = errors.New("offline") },
			handler:  &fakeHandler{status: errored},
			assert: func(t *testing.T, _ *fakeStore, _ *fakeHandler, _ *nats.Subscription, err error) {
				require.ErrorContains(t, err, `error updating status of PromotionRequest "dev.01j" in Project "demo": offline`)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			conn := connect(t)
			sub, err := conn.SubscribeSync(SubjectUpdated)
			require.NoError(t, err)
			store := newFakeStore(testCase.snapshot)
			if testCase.store != nil {
				testCase.store(store)
			}
			r := &reconciler{store: store, handler: testCase.handler, conn: conn}
			_, err = r.Reconcile(t.Context(), dbreconcile.Request[uuid.UUID]{Key: testCase.snapshot.ID})
			testCase.assert(t, store, testCase.handler, sub, err)
		})
	}
}

// TestController runs the reconciler as the controller binary does, over a
// real NATS connection.
func TestController(t *testing.T) {
	t.Parallel()

	start := func(t *testing.T, store *fakeStore, conn *nats.Conn, interval time.Duration) {
		t.Helper()
		mgr := &fakeManager{}
		require.NoError(t, setup(mgr, store, NewEnterpriseOnlyHandler(), conn, ReconcilerConfig{
			Interval:                interval,
			ResyncPageSize:          10,
			MaxConcurrentReconciles: 2,
		}))
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() { done <- mgr.added[0].Start(ctx) }()
		t.Cleanup(func() {
			cancel()
			require.NoError(t, <-done)
		})
	}

	t.Run("a new request is reconciled when it is announced", func(t *testing.T) {
		t.Parallel()
		conn := connect(t)
		store := newFakeStore()
		// A long interval: only the announcement can cause the reconcile.
		start(t, store, conn, time.Hour)
		require.Eventually(t, func() bool { return conn.NumSubscriptions() == 1 }, time.Second, time.Millisecond)

		snapshot := testSnapshot(kargoapi.PromotionRequestPhasePending)
		store.mu.Lock()
		store.requests[snapshot.ID] = snapshot
		store.mu.Unlock()
		require.NoError(t, PublishCreated(conn, snapshot))

		require.Eventually(t, func() bool {
			return store.phase(snapshot.ID) == string(kargoapi.PromotionRequestPhaseErrored)
		}, 2*time.Second, time.Millisecond)
		// Its own announcement of the write did not cause another.
		require.Never(t, func() bool { return store.updateCount() > 1 }, 100*time.Millisecond, 5*time.Millisecond)
	})

	t.Run("an unannounced request is found by the resync", func(t *testing.T) {
		t.Parallel()
		snapshot := testSnapshot(kargoapi.PromotionRequestPhasePending)
		store := newFakeStore(snapshot, testSnapshot(kargoapi.PromotionRequestPhaseFailed))
		start(t, store, connect(t), 20*time.Millisecond)
		require.Eventually(t, func() bool {
			return store.phase(snapshot.ID) == string(kargoapi.PromotionRequestPhaseErrored)
		}, 2*time.Second, time.Millisecond)
		// The finished one was never listed, so only the open one was written.
		require.Never(t, func() bool { return store.updateCount() > 1 }, 100*time.Millisecond, 5*time.Millisecond)
	})
}
