package dbreconcile

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// fakeSource is a Source the test drives by hand.
type fakeSource[K comparable] struct {
	startErr   error
	started    chan struct{}
	ctx        context.Context
	emit       func(context.Context, Event[K, item])
	queueGroup string
}

func newFakeSource[K comparable]() *fakeSource[K] {
	return &fakeSource[K]{started: make(chan struct{})}
}

func (s *fakeSource[K]) Start(
	ctx context.Context,
	emit func(context.Context, Event[K, item]),
	queueGroup string,
) error {
	if s.startErr != nil {
		return s.startErr
	}
	s.ctx, s.emit, s.queueGroup = ctx, emit, queueGroup
	close(s.started)
	return nil
}

func (s *fakeSource[K]) waitStarted(t *testing.T) {
	t.Helper()
	select {
	case <-s.started:
	case <-time.After(time.Second):
		t.Fatal("source was never started")
	}
}

func (s *fakeSource[K]) send(t *testing.T, e Event[K, item]) {
	t.Helper()
	s.waitStarted(t)
	s.emit(s.ctx, e)
}

// group returns the queue group the source was started with.
func (s *fakeSource[K]) group(t *testing.T) string {
	t.Helper()
	s.waitStarted(t)
	return s.queueGroup
}

// recorder counts reconciles per key.
type recorder struct {
	mu    sync.Mutex
	calls map[string]int
}

func (r *recorder) record(key string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.calls == nil {
		r.calls = make(map[string]int)
	}
	r.calls[key]++
	return r.calls[key]
}

func (r *recorder) count(key string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls[key]
}

func (r *recorder) keys() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	keys := make([]string, 0, len(r.calls))
	for key := range r.calls {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

type fakeManager struct {
	added []manager.Runnable
	err   error
}

func (m *fakeManager) Add(r manager.Runnable) error {
	if m.err != nil {
		return m.err
	}
	m.added = append(m.added, r)
	return nil
}

func fastRateLimiter() workqueue.TypedRateLimiter[Request[string]] {
	return workqueue.NewTypedItemExponentialFailureRateLimiter[Request[string]](time.Millisecond, 10*time.Millisecond)
}

// builder returns a Builder for string keys with a fast rate limiter.
func builder() *Builder[string] {
	return NewControllerManagedBy[string](&fakeManager{}).
		Named("items").
		WithRateLimiter(fastRateLimiter())
}

// run starts the controller and stops it when the test ends, requiring that
// it stops cleanly.
func run(t *testing.T, b *Builder[string], r Reconciler[string]) {
	t.Helper()
	c, err := b.Build(r)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- c.Start(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(2 * time.Second):
			t.Error("controller did not stop")
		}
	})
}

func receive(t *testing.T, ch <-chan string) string {
	t.Helper()
	select {
	case key := <-ch:
		return key
	case <-time.After(2 * time.Second):
		t.Fatal("nothing was reconciled")
		return ""
	}
}

func never[K comparable](Event[K, item]) bool { return false }

func TestBuild(t *testing.T) {
	t.Parallel()
	reconciler := ReconcilerFunc[string](func(context.Context, Request[string]) (Result, error) {
		return Result{}, nil
	})
	lister := ListerFunc[string](func(context.Context, string, int) ([]string, error) { return nil, nil })
	watch := NewWatch(newFakeSource[string](), EnqueueKey[string, item]())
	testCases := []struct {
		name       string
		builder    *Builder[string]
		reconciler Reconciler[string]
		assert     func(*testing.T, *Controller[string], error)
	}{
		{
			name:       "name is required",
			builder:    NewControllerManagedBy[string](nil).Watches(watch),
			reconciler: reconciler,
			assert: func(t *testing.T, _ *Controller[string], err error) {
				require.EqualError(t, err, "controller name is required")
			},
		},
		{
			name:       "name must not contain whitespace",
			builder:    NewControllerManagedBy[string](nil).Named("my items").Watches(watch),
			reconciler: reconciler,
			assert: func(t *testing.T, _ *Controller[string], err error) {
				require.EqualError(t, err, `controller name "my items" must not contain whitespace`)
			},
		},
		{
			name:    "reconciler is required",
			builder: NewControllerManagedBy[string](nil).Named("items").Watches(watch),
			assert: func(t *testing.T, _ *Controller[string], err error) {
				require.EqualError(t, err, `controller "items" requires a reconciler`)
			},
		},
		{
			name:       "something must cause reconciles",
			builder:    NewControllerManagedBy[string](nil).Named("items"),
			reconciler: reconciler,
			assert: func(t *testing.T, _ *Controller[string], err error) {
				require.EqualError(t, err, `controller "items" watches nothing and never resyncs`)
			},
		},
		{
			name:       "resync needs an interval",
			builder:    NewControllerManagedBy[string](nil).Named("items").Resync(lister, 0),
			reconciler: reconciler,
			assert: func(t *testing.T, _ *Controller[string], err error) {
				require.EqualError(t, err, `controller "items" requires a positive resync interval`)
			},
		},
		{
			name: "resync needs a page size",
			builder: NewControllerManagedBy[string](nil).Named("items").
				Resync(lister, time.Minute).WithResyncPageSize(0),
			reconciler: reconciler,
			assert: func(t *testing.T, _ *Controller[string], err error) {
				require.EqualError(t, err, `controller "items" requires a positive resync page size`)
			},
		},
		{
			name:       "at least one worker",
			builder:    NewControllerManagedBy[string](nil).Named("items").Watches(watch).WithWorkers(0),
			reconciler: reconciler,
			assert: func(t *testing.T, _ *Controller[string], err error) {
				require.EqualError(t, err, `controller "items" requires at least one worker`)
			},
		},
		{
			name:       "defaults",
			builder:    NewControllerManagedBy[string](nil).Named("items").Watches(watch),
			reconciler: reconciler,
			assert: func(t *testing.T, c *Controller[string], err error) {
				require.NoError(t, err)
				require.Equal(t, "items", c.name)
				require.Equal(t, defaultWorkers, c.workers)
				require.Equal(t, defaultResyncPageSize, c.resyncPageSize)
				require.NotNil(t, c.rateLimiter)
				require.Nil(t, c.lister)
				require.Len(t, c.watches, 1)
			},
		},
		{
			name: "resync alone is enough",
			builder: NewControllerManagedBy[string](nil).Named("items").
				Resync(lister, time.Minute).WithResyncPageSize(7).WithWorkers(3),
			reconciler: reconciler,
			assert: func(t *testing.T, c *Controller[string], err error) {
				require.NoError(t, err)
				require.Equal(t, time.Minute, c.resyncInterval)
				require.Equal(t, 7, c.resyncPageSize)
				require.Equal(t, 3, c.workers)
				require.Empty(t, c.watches)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			c, err := testCase.builder.Build(testCase.reconciler)
			testCase.assert(t, c, err)
		})
	}
}

func TestComplete(t *testing.T) {
	t.Parallel()
	reconciler := ReconcilerFunc[string](func(context.Context, Request[string]) (Result, error) {
		return Result{}, nil
	})
	watch := NewWatch(newFakeSource[string](), EnqueueKey[string, item]())

	require.EqualError(
		t,
		NewControllerManagedBy[string](nil).Named("items").Watches(watch).Complete(reconciler),
		"a manager is required",
	)
	require.EqualError(
		t,
		NewControllerManagedBy[string](&fakeManager{}).Watches(watch).Complete(reconciler),
		"controller name is required",
	)
	require.EqualError(
		t,
		NewControllerManagedBy[string](&fakeManager{err: errors.New("stopped")}).
			Named("items").Watches(watch).Complete(reconciler),
		`error adding controller "items" to manager: stopped`,
	)

	mgr := &fakeManager{}
	require.NoError(t, NewControllerManagedBy[string](mgr).Named("items").Watches(watch).Complete(reconciler))
	require.Len(t, mgr.added, 1)
	c, ok := mgr.added[0].(*Controller[string])
	require.True(t, ok)
	require.False(t, c.NeedLeaderElection())
}

func TestWatch(t *testing.T) {
	t.Parallel()

	t.Run("a source is required", func(t *testing.T) {
		t.Parallel()
		w := NewWatch[string, item, string](nil, EnqueueKey[string, item]())
		require.EqualError(t, w.start(t.Context(), func(string) {}, ""), "a watch requires a source")
	})

	t.Run("a handler is required", func(t *testing.T) {
		t.Parallel()
		w := NewWatch[string, item, string](newFakeSource[string](), nil)
		require.EqualError(t, w.start(t.Context(), func(string) {}, ""), "a watch requires an event handler")
	})

	t.Run("an event must satisfy every predicate", func(t *testing.T) {
		t.Parallel()
		source := newFakeSource[string]()
		w := NewWatch(
			source,
			EnqueueKey[string, item](),
			Predicate[string, item]{Update: func(e Event[string, item]) bool { return e.New.Phase != "" }},
			Predicate[string, item]{Update: func(e Event[string, item]) bool { return e.Old.Phase != e.New.Phase }},
		)
		var enqueued []string
		require.NoError(t, w.start(t.Context(), func(key string) { enqueued = append(enqueued, key) }, ""))
		source.send(t, updated("a", "Pending", "Running")) // both
		source.send(t, updated("b", "", ""))               // neither
		source.send(t, updated("c", "Running", "Running")) // only the first
		source.send(t, created("d"))                       // no Create function: allowed
		require.Equal(t, []string{"a", "d"}, enqueued)
	})
}

func TestControllerReconcilesWhatItWatches(t *testing.T) {
	t.Parallel()
	own := newFakeSource[string]()
	// Another resource, keyed by number, whose events map to this
	// controller's keys.
	other := newFakeSource[int]()
	rec := &recorder{}
	reconciled := make(chan string, 10)
	run(
		t,
		builder().
			Watches(NewWatch(
				own,
				EnqueueKey[string, item](),
				Predicate[string, item]{Update: never[string]},
			)).
			Watches(NewWatch(
				other,
				EnqueueMapped(func(_ context.Context, e Event[int, item]) []string {
					return []string{"parent-of-" + strconv.Itoa(e.Key)}
				}),
			)),
		ReconcilerFunc[string](func(_ context.Context, req Request[string]) (Result, error) {
			rec.record(req.Key)
			reconciled <- req.Key
			return Result{}, nil
		}),
	)

	// Each watch shares its events through a group of its own.
	require.Equal(t, "items.0", own.group(t))
	require.Equal(t, "items.1", other.group(t))
	own.send(t, updated("filtered", "Pending", "Running"))
	own.send(t, created("a"))
	other.send(t, Event[int, item]{Kind: Created, Key: 7, New: &item{}})
	receive(t, reconciled)
	receive(t, reconciled)
	require.Never(t, func() bool { return len(reconciled) > 0 }, 50*time.Millisecond, 5*time.Millisecond)
	require.Equal(t, []string{"a", "parent-of-7"}, rec.keys())
}

func TestControllerDeduplicatesKeys(t *testing.T) {
	t.Parallel()
	source := newFakeSource[string]()
	rec := &recorder{}
	entered := make(chan string, 10)
	release := make(chan struct{})
	run(
		t,
		builder().Watches(NewWatch(source, EnqueueKey[string, item]())),
		ReconcilerFunc[string](func(ctx context.Context, req Request[string]) (Result, error) {
			n := rec.record(req.Key)
			entered <- req.Key
			if n == 1 {
				select {
				case <-release:
				case <-ctx.Done():
				}
			}
			return Result{}, nil
		}),
	)

	source.send(t, created("a"))
	receive(t, entered)
	// Events arriving while the key is being reconciled fold into one more
	// reconcile, not one per event.
	for range 5 {
		source.send(t, updated("a", "Pending", "Running"))
	}
	close(release)
	receive(t, entered)
	require.Never(t, func() bool { return rec.count("a") > 2 }, 100*time.Millisecond, 5*time.Millisecond)
}

func TestControllerRetriesFailures(t *testing.T) {
	t.Parallel()

	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		source := newFakeSource[string]()
		rec := &recorder{}
		run(
			t,
			builder().Watches(NewWatch(source, EnqueueKey[string, item]())),
			ReconcilerFunc[string](func(_ context.Context, req Request[string]) (Result, error) {
				if rec.record(req.Key) < 3 {
					return Result{}, errors.New("not yet")
				}
				return Result{}, nil
			}),
		)
		source.send(t, created("failing"))
		require.Eventually(t, func() bool { return rec.count("failing") == 3 }, 2*time.Second, time.Millisecond)
		require.Never(t, func() bool { return rec.count("failing") > 3 }, 100*time.Millisecond, 5*time.Millisecond)
	})

	t.Run("panics", func(t *testing.T) {
		t.Parallel()
		source := newFakeSource[string]()
		rec := &recorder{}
		run(
			t,
			builder().Watches(NewWatch(source, EnqueueKey[string, item]())),
			ReconcilerFunc[string](func(_ context.Context, req Request[string]) (Result, error) {
				if rec.record(req.Key) == 1 {
					panic("boom")
				}
				return Result{}, nil
			}),
		)
		source.send(t, created("panicking"))
		require.Eventually(t, func() bool { return rec.count("panicking") == 2 }, 2*time.Second, time.Millisecond)
	})
}

func TestControllerReconcilePanicBecomesError(t *testing.T) {
	t.Parallel()
	c, err := builder().
		Watches(NewWatch(newFakeSource[string](), EnqueueKey[string, item]())).
		Build(ReconcilerFunc[string](func(context.Context, Request[string]) (Result, error) {
			panic("boom")
		}))
	require.NoError(t, err)
	_, err = c.reconcile(t.Context(), Request[string]{Key: "a"})
	require.EqualError(t, err, "panic reconciling key a: boom")
}

func TestControllerRequeuesAfter(t *testing.T) {
	t.Parallel()
	source := newFakeSource[string]()
	rec := &recorder{}
	run(
		t,
		builder().Watches(NewWatch(source, EnqueueKey[string, item]())),
		ReconcilerFunc[string](func(_ context.Context, req Request[string]) (Result, error) {
			if rec.record(req.Key) == 1 {
				return Result{RequeueAfter: 10 * time.Millisecond}, nil
			}
			return Result{}, nil
		}),
	)
	source.send(t, created("requeued"))
	require.Eventually(t, func() bool { return rec.count("requeued") == 2 }, 2*time.Second, time.Millisecond)
	require.Never(t, func() bool { return rec.count("requeued") > 2 }, 100*time.Millisecond, 5*time.Millisecond)
}

func TestControllerWorkers(t *testing.T) {
	t.Parallel()
	source := newFakeSource[string]()
	entered := make(chan string, 10)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	run(
		t,
		builder().Watches(NewWatch(source, EnqueueKey[string, item]())).WithWorkers(2),
		ReconcilerFunc[string](func(ctx context.Context, req Request[string]) (Result, error) {
			entered <- req.Key
			select {
			case <-release:
			case <-ctx.Done():
			}
			return Result{}, nil
		}),
	)
	source.send(t, created("a"))
	source.send(t, created("b"))
	// Both are in their Reconcile at once.
	got := []string{receive(t, entered), receive(t, entered)}
	slices.Sort(got)
	require.Equal(t, []string{"a", "b"}, got)
}

func TestControllerResync(t *testing.T) {
	t.Parallel()
	all := []string{"a", "b", "c"}
	var lists atomic.Int32
	lister := ListerFunc[string](func(_ context.Context, after string, limit int) ([]string, error) {
		lists.Add(1)
		var page []string
		for _, key := range all {
			if key > after && len(page) < limit {
				page = append(page, key)
			}
		}
		return page, nil
	})
	rec := &recorder{}
	run(
		t,
		builder().
			// A watch whose predicate lets nothing through: only the resync
			// can cause these reconciles.
			Watches(NewWatch(
				newFakeSource[string](),
				EnqueueKey[string, item](),
				Predicate[string, item]{Create: never[string], Update: never[string], Delete: never[string]},
			)).
			Resync(lister, 20*time.Millisecond).
			WithResyncPageSize(2),
		ReconcilerFunc[string](func(_ context.Context, req Request[string]) (Result, error) {
			rec.record(req.Key)
			return Result{}, nil
		}),
	)
	require.Eventually(t, func() bool { return len(rec.keys()) == 3 }, 2*time.Second, time.Millisecond)
	require.Equal(t, all, rec.keys())
	// Every resync pages: {a, b}, then {c}. It runs again on every tick.
	require.Eventually(t, func() bool { return lists.Load() >= 6 }, 2*time.Second, time.Millisecond)
}

func TestControllerResyncFailures(t *testing.T) {
	t.Parallel()
	newController := func(lister ListerFunc[string]) *Controller[string] {
		c, err := builder().Resync(lister, time.Minute).WithResyncPageSize(2).
			Build(ReconcilerFunc[string](func(context.Context, Request[string]) (Result, error) {
				return Result{}, nil
			}))
		require.NoError(t, err)
		return c
	}

	t.Run("a failed list", func(t *testing.T) {
		t.Parallel()
		c := newController(func(context.Context, string, int) ([]string, error) {
			return nil, errors.New("offline")
		})
		require.EqualError(t, c.resync(t.Context(), func(string) {}), "error listing keys: offline")
	})

	t.Run("a full page that does not advance", func(t *testing.T) {
		t.Parallel()
		// A lister that ignores after returns the same page forever.
		c := newController(func(context.Context, string, int) ([]string, error) {
			return []string{"a", "b"}, nil
		})
		var enqueued []string
		err := c.resync(t.Context(), func(key string) { enqueued = append(enqueued, key) })
		require.EqualError(t, err, "lister returned a full page without advancing past key b")
		require.Equal(t, []string{"a", "b", "a", "b"}, enqueued)
	})
}

func TestControllerStartFailure(t *testing.T) {
	t.Parallel()
	first := newFakeSource[string]()
	failing := newFakeSource[string]()
	failing.startErr = errors.New("cannot subscribe")
	c, err := builder().
		Watches(NewWatch(first, EnqueueKey[string, item]())).
		Watches(NewWatch(failing, EnqueueKey[string, item]())).
		Build(ReconcilerFunc[string](func(context.Context, Request[string]) (Result, error) {
			return Result{}, nil
		}))
	require.NoError(t, err)
	require.EqualError(
		t,
		c.Start(t.Context()),
		`error starting watch 1 of controller "items": cannot subscribe`,
	)
	// The watch that did start was told to stop.
	select {
	case <-first.ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("started watch was not stopped")
	}
}

func TestControllerOverNATS(t *testing.T) {
	t.Parallel()
	conn := connect(t)
	reconciled := make(chan string, 10)
	run(
		t,
		builder().Watches(NewWatch(
			Subject[string, item](conn, "kargo.items.>"),
			EnqueueKey[string, item](),
			Predicate[string, item]{
				Update: func(e Event[string, item]) bool { return e.Old.Phase != e.New.Phase },
			},
		)),
		ReconcilerFunc[string](func(_ context.Context, req Request[string]) (Result, error) {
			reconciled <- req.Key
			return Result{}, nil
		}),
	)
	// The subscription is set up before Start's watches return, but Start
	// runs in the background; wait for it.
	require.Eventually(t, func() bool { return conn.NumSubscriptions() == 1 }, time.Second, time.Millisecond)

	require.NoError(t, Publish(conn, "kargo.items.updated", updated("unchanged", "Running", "Running")))
	require.NoError(t, Publish(conn, "kargo.items.updated", updated("a", "Pending", "Running")))
	require.Equal(t, "a", receive(t, reconciled))
	require.Never(t, func() bool { return len(reconciled) > 0 }, 50*time.Millisecond, 5*time.Millisecond)
}

func TestControllerReplicasOverNATS(t *testing.T) {
	t.Parallel()

	// replicas starts two controllers of the same name, each with two watches
	// on the same subject, as two replicas of one controller would be. Each
	// watch tags the key with its position, so a reconcile says which watch
	// caused it.
	replicas := func(t *testing.T, conn *nats.Conn, rec *recorder, opts ...SubjectOption) {
		for range 2 {
			b := builder()
			for i := range 2 {
				b = b.Watches(NewWatch(
					Subject[string, item](conn, "kargo.items.>", opts...),
					EnqueueMapped(func(_ context.Context, e Event[string, item]) []string {
						return []string{e.Key + "/" + strconv.Itoa(i)}
					}),
				))
			}
			run(t, b, ReconcilerFunc[string](func(_ context.Context, req Request[string]) (Result, error) {
				rec.record(req.Key)
				return Result{}, nil
			}))
		}
		require.Eventually(t, func() bool { return conn.NumSubscriptions() == 4 }, time.Second, time.Millisecond)
	}
	publish := func(t *testing.T, conn *nats.Conn) []string {
		var want []string
		for i := range 5 {
			key := strconv.Itoa(i)
			require.NoError(t, Publish(conn, "kargo.items.created", created(key)))
			want = append(want, key+"/0", key+"/1")
		}
		slices.Sort(want)
		return want
	}
	total := func(rec *recorder) int {
		n := 0
		for _, key := range rec.keys() {
			n += rec.count(key)
		}
		return n
	}

	t.Run("replicas share events, and each watch sees all of them", func(t *testing.T) {
		t.Parallel()
		conn := connect(t)
		rec := &recorder{}
		replicas(t, conn, rec)
		want := publish(t, conn)
		// Every event reaches both watches, and each watch's copy reaches
		// exactly one replica.
		require.Eventually(t, func() bool { return slices.Equal(want, rec.keys()) }, 2*time.Second, time.Millisecond)
		require.Never(t, func() bool { return total(rec) > len(want) }, 100*time.Millisecond, 5*time.Millisecond)
	})

	t.Run("without a queue group, every replica receives every event", func(t *testing.T) {
		t.Parallel()
		conn := connect(t)
		rec := &recorder{}
		replicas(t, conn, rec, WithoutQueueGroup())
		want := publish(t, conn)
		require.Eventually(t, func() bool { return total(rec) == 2*len(want) }, 2*time.Second, time.Millisecond)
		require.Equal(t, want, rec.keys())
	})
}
