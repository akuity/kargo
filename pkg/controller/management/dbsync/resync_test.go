package dbsync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/management/dbsync/internal/syncapi"
)

func TestResync(t *testing.T) {
	t.Parallel()
	changes := syncapi.Changes{
		ToSync:   []client.ObjectKey{{Namespace: "demo", Name: "dev"}, {Namespace: "other", Name: "dev"}},
		ToDelete: []string{"old-uid"},
	}
	testCases := []struct {
		name     string
		changes  syncapi.Changes
		diffErr  error
		pruneErr error
		wantErr  string
	}{
		{name: "matching database generates no work"},
		{name: "only differences are queued or pruned", changes: changes},
		{
			name: "failed diff ignores all returned changes", changes: changes,
			diffErr: errors.New("incomplete list"), wantErr: "incomplete list",
		},
		{
			name: "cleanup error propagates", changes: changes,
			pruneErr: errors.New("offline"), wantErr: "offline",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			s := &fakeSyncer{changes: testCase.changes, diffErr: testCase.diffErr, pruneErr: testCase.pruneErr}
			events := make(chan event.GenericEvent, 2)
			r := &resyncRunner{targets: []resyncTarget{{name: "stage", syncer: s, events: events}}}
			err := r.resync(context.Background())
			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
			}
			if testCase.diffErr != nil {
				require.Empty(t, events)
				require.Empty(t, s.pruned)
			} else {
				require.Len(t, events, len(testCase.changes.ToSync))
				for _, key := range testCase.changes.ToSync {
					obj := (<-events).Object
					require.IsType(t, &kargoapi.Stage{}, obj)
					require.Equal(t, key, client.ObjectKeyFromObject(obj))
					require.Empty(t, obj.GetUID())
				}
				require.Equal(t, testCase.changes.ToDelete, s.pruned)
			}
			require.Empty(t, s.synced) // Upserts always go through the controller queue.
			require.Empty(t, s.deleted)
		})
	}
}

func TestResyncIndependentResources(t *testing.T) {
	t.Parallel()
	failed := &fakeSyncer{diffErr: errors.New("projects unavailable")}
	stage := &fakeSyncer{changes: syncapi.Changes{ToSync: []client.ObjectKey{{Namespace: "demo", Name: "dev"}}}}
	projectEvents := make(chan event.GenericEvent, 1)
	stageEvents := make(chan event.GenericEvent, 1)
	r := &resyncRunner{targets: []resyncTarget{
		{name: "project", syncer: failed, events: projectEvents},
		{name: "stage", syncer: stage, events: stageEvents},
	}}
	require.ErrorContains(t, r.resync(context.Background()), "projects unavailable")
	require.Empty(t, projectEvents)
	require.Len(t, stageEvents, 1)
}

func TestResyncCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := &fakeSyncer{changes: syncapi.Changes{
		ToSync: []client.ObjectKey{{Name: "demo"}}, ToDelete: []string{"old"},
	}}
	target := resyncTarget{syncer: s, events: make(chan event.GenericEvent)}
	require.ErrorIs(t, target.resync(ctx), context.Canceled)
	require.Empty(t, s.pruned)
}

func TestResyncStart(t *testing.T) {
	t.Parallel()
	for _, failFirst := range []bool{false, true} {
		name := map[bool]string{false: "startup and periodic", true: "recovers after failure"}[failFirst]
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			events := make(chan event.GenericEvent, 2)
			s := &fakeSyncer{
				failFirstDiff: failFirst,
				changes:       syncapi.Changes{ToSync: []client.ObjectKey{{Namespace: "demo", Name: "dev"}}},
			}
			r := &resyncRunner{
				cache: &syncedCache{ready: true}, interval: 10 * time.Millisecond,
				targets: []resyncTarget{{name: "stage", syncer: s, events: events}},
			}
			done := make(chan error, 1)
			go func() { done <- r.Start(ctx) }()
			for range 2 {
				select {
				case <-events:
				case <-time.After(5 * time.Second):
					t.Fatal("resync did not enqueue stages")
				}
			}
			cancel()
			select {
			case err := <-done:
				require.NoError(t, err)
			case <-time.After(5 * time.Second):
				t.Fatal("resync did not stop")
			}
			require.GreaterOrEqual(t, s.diffCalls, 2)
		})
	}
}

func TestResyncRequiresCacheSync(t *testing.T) {
	t.Parallel()
	s := &fakeSyncer{}
	r := &resyncRunner{cache: &syncedCache{}, targets: []resyncTarget{{syncer: s}}}
	require.NoError(t, r.Start(context.Background()))
	require.Zero(t, s.diffCalls)
}

type syncedCache struct{ ready bool }

func (c *syncedCache) WaitForCacheSync(context.Context) bool { return c.ready }
