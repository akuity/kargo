package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/akuity/kargo/pkg/logging"
)

// fakeManager is a manager.Manager whose Start method is replaced by startFn.
// Calling any other method panics.
type fakeManager struct {
	manager.Manager
	startFn func(context.Context) error
}

func (m *fakeManager) Start(ctx context.Context) error {
	return m.startFn(ctx)
}

// cancelTiming says when a test cancels the context passed to startManagers.
type cancelTiming int

const (
	// cancelNever leaves the context alone until startManagers has returned.
	cancelNever cancelTiming = iota
	// cancelBeforeStart cancels the context before startManagers is called.
	cancelBeforeStart
	// cancelWhenRunning cancels the context once the managers' caches have
	// synced. It requires real managers.
	cancelWhenRunning
)

func TestStartManagers(t *testing.T) {
	testErr := errors.New("something went wrong")

	// newManager returns a real manager. Its cache has no informers, so it never
	// needs to reach the (nonexistent) API server it is configured with.
	newManager := func(t *testing.T) manager.Manager {
		mgr, err := manager.New(
			&rest.Config{Host: "http://127.0.0.1:1"},
			manager.Options{
				Metrics:                metricsserver.Options{BindAddress: "0"},
				HealthProbeBindAddress: "0",
			},
		)
		require.NoError(t, err)
		return mgr
	}
	// failingManager returns a manager that fails as soon as it is started.
	failingManager := func(*testing.T) manager.Manager {
		return &fakeManager{
			startFn: func(context.Context) error { return testErr },
		}
	}
	// stoppedManager returns a manager that stops without error as soon as it
	// is started.
	stoppedManager := func(*testing.T) manager.Manager {
		return &fakeManager{
			startFn: func(context.Context) error { return nil },
		}
	}
	// runningManager returns a manager that runs until its context is done.
	runningManager := func(*testing.T) manager.Manager {
		return &fakeManager{
			startFn: func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			},
		}
	}

	testCases := []struct {
		name      string
		kargoMgr  func(*testing.T) manager.Manager
		argocdMgr func(*testing.T) manager.Manager
		cancel    cancelTiming
		assert    func(*testing.T, error)
	}{
		{
			name:     "graceful shutdown without Argo CD manager",
			kargoMgr: newManager,
			cancel:   cancelWhenRunning,
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:      "graceful shutdown with Argo CD manager",
			kargoMgr:  newManager,
			argocdMgr: newManager,
			cancel:    cancelWhenRunning,
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:      "context canceled before start",
			kargoMgr:  newManager,
			argocdMgr: newManager,
			cancel:    cancelBeforeStart,
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:      "Kargo manager fails while Argo CD manager runs",
			kargoMgr:  failingManager,
			argocdMgr: runningManager,
			cancel:    cancelNever,
			assert: func(t *testing.T, err error) {
				require.ErrorIs(t, err, testErr)
				require.ErrorContains(t, err, "error starting Kargo manager")
			},
		},
		{
			name:      "Argo CD manager fails while Kargo manager runs",
			kargoMgr:  runningManager,
			argocdMgr: failingManager,
			cancel:    cancelNever,
			assert: func(t *testing.T, err error) {
				require.ErrorIs(t, err, testErr)
				require.ErrorContains(t, err, "error starting Argo CD manager")
			},
		},
		{
			name:      "Kargo manager fails after Argo CD manager stops",
			kargoMgr:  failingManager,
			argocdMgr: stoppedManager,
			cancel:    cancelNever,
			assert: func(t *testing.T, err error) {
				require.ErrorIs(t, err, testErr)
				require.ErrorContains(t, err, "error starting Kargo manager")
			},
		},
		{
			name:      "both managers fail",
			kargoMgr:  failingManager,
			argocdMgr: failingManager,
			cancel:    cancelNever,
			assert: func(t *testing.T, err error) {
				require.ErrorIs(t, err, testErr)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			// Some of the behavior under test depends on goroutine scheduling,
			// so repeat each case to make a regression very unlikely to slip by.
			for range 20 {
				kargoMgr := testCase.kargoMgr(t)
				var argocdMgr manager.Manager
				if testCase.argocdMgr != nil {
					argocdMgr = testCase.argocdMgr(t)
				}

				ctx, cancel := context.WithCancel(t.Context())
				if testCase.cancel == cancelBeforeStart {
					cancel()
				}
				errCh := make(chan error, 1)
				go func() {
					o := &controllerOptions{Logger: logging.NewDiscardLoggerOrDie()}
					errCh <- o.startManagers(ctx, kargoMgr, argocdMgr)
				}()

				if testCase.cancel == cancelWhenRunning {
					// Wait for the managers to be up and running before stopping
					// them.
					waitCtx, waitCancel := context.WithTimeout(ctx, 10*time.Second)
					require.True(t, kargoMgr.GetCache().WaitForCacheSync(waitCtx))
					if argocdMgr != nil {
						require.True(t, argocdMgr.GetCache().WaitForCacheSync(waitCtx))
					}
					waitCancel()
					cancel()
				}

				select {
				case err := <-errCh:
					testCase.assert(t, err)
				case <-time.After(10 * time.Second):
					require.FailNow(t, "timed out waiting for startManagers to return")
				}
				cancel()
			}
		})
	}
}
