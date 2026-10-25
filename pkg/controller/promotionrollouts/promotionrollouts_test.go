package promotionrollouts

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/reconciler"
	"github.com/akuity/kargo/pkg/reconciler/list"
)

// fakeStore records every sweep it is asked for, and answers each with the
// number of rollouts it ended, or fails.
type fakeStore struct {
	mu     sync.Mutex
	ended  int64
	err    error
	sweeps []struct{ reason, message string }
}

func (s *fakeStore) ErrorPending(_ context.Context, reason, message string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweeps = append(s.sweeps, struct{ reason, message string }{reason, message})
	if s.err != nil {
		return 0, s.err
	}
	return s.ended, nil
}

func (s *fakeStore) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sweeps)
}

func TestReconcile(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		store      *fakeStore
		assertions func(*testing.T, *fakeStore, reconciler.Result, error)
	}{
		{
			name:  "ends what has not started as an enterprise-only feature",
			store: &fakeStore{ended: 3},
			assertions: func(t *testing.T, s *fakeStore, result reconciler.Result, err error) {
				require.NoError(t, err)
				require.Zero(t, result)
				require.Equal(t, []struct{ reason, message string }{
					{enterpriseOnlyReason, enterpriseOnlyMessage},
				}, s.sweeps)
			},
		},
		{
			name:  "nothing to end is not an error",
			store: &fakeStore{},
			assertions: func(t *testing.T, s *fakeStore, result reconciler.Result, err error) {
				require.NoError(t, err)
				require.Zero(t, result)
				require.Equal(t, 1, s.calls())
			},
		},
		{
			name:  "a failed sweep is retried",
			store: &fakeStore{err: errors.New("database is down")},
			assertions: func(t *testing.T, _ *fakeStore, _ reconciler.Result, err error) {
				require.ErrorContains(t, err, "database is down")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result, err := newReconciler(testCase.store)(t.Context(), sweep{})
			testCase.assertions(t, testCase.store, result, err)
		})
	}
}

func TestSweeps(t *testing.T) {
	t.Parallel()
	requests, err := sweeps(t.Context())
	require.NoError(t, err)
	require.Equal(t, []sweep{{}}, requests)
}

func TestControllerSweepsOnStart(t *testing.T) {
	t.Parallel()
	s := &fakeStore{ended: 1}
	c, err := reconciler.New[sweep](controllerName).
		Watch(list.New(sweeps)).
		Func(newReconciler(s))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- c.Start(ctx) }()
	require.Eventually(t, func() bool { return s.calls() == 1 }, 5*time.Second, 10*time.Millisecond)
	cancel()
	select {
	case err = <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("controller did not stop")
	}
}

func TestReconcilerConfigFromEnv(t *testing.T) {
	cfg := ReconcilerConfigFromEnv()
	require.Equal(t, time.Minute, cfg.ResyncInterval)

	t.Setenv("PROMOTION_ROLLOUT_RESYNC_INTERVAL", "30s")
	cfg = ReconcilerConfigFromEnv()
	require.Equal(t, 30*time.Second, cfg.ResyncInterval)
}
