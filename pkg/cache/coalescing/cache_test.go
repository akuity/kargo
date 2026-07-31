package coalescing

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
)

const (
	testKey         = "test-key"
	testInput       = "test-input"
	testValue       = "test-value"
	testLoadTimeout = 30 * time.Second
)

func TestNewCache(t *testing.T) {
	t.Parallel()

	zero := time.Duration(0)
	negative := -1 * time.Minute
	tenMinutes := 10 * time.Minute

	testCases := []struct {
		name       string
		opts       *CacheOptions
		assertions func(*testing.T, *CacheOptions, Cache[string, string], error)
	}{
		{
			name: "nil CacheOptions",
			assertions: func(
				t *testing.T,
				_ *CacheOptions,
				c Cache[string, string],
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, defaultLoadTimeout, loadTimeoutOf(c))
				require.NotNil(t, internalValues(c))
			},
		},
		{
			name: "caller's CacheOptions are left as they were found",
			opts: &CacheOptions{},
			assertions: func(
				t *testing.T,
				opts *CacheOptions,
				_ Cache[string, string],
				err error,
			) {
				require.NoError(t, err)
				require.Nil(t, opts.CleanupInterval)
				require.Nil(t, opts.DefaultTTL)
				require.Nil(t, opts.LoadTimeout)
			},
		},
		{
			name: "explicit LoadTimeout",
			opts: &CacheOptions{LoadTimeout: &tenMinutes},
			assertions: func(
				t *testing.T,
				_ *CacheOptions,
				c Cache[string, string],
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, tenMinutes, loadTimeoutOf(c))
			},
		},
		{
			name: "zero LoadTimeout",
			opts: &CacheOptions{LoadTimeout: &zero},
			assertions: func(
				t *testing.T,
				_ *CacheOptions,
				c Cache[string, string],
				err error,
			) {
				require.ErrorContains(t, err, "LoadTimeout must be positive")
				require.Nil(t, c)
			},
		},
		{
			name: "negative LoadTimeout",
			opts: &CacheOptions{LoadTimeout: &negative},
			assertions: func(
				t *testing.T,
				_ *CacheOptions,
				c Cache[string, string],
				err error,
			) {
				require.ErrorContains(t, err, "LoadTimeout must be positive")
				require.Nil(t, c)
			},
		},
		{
			name: "zero CleanupInterval",
			opts: &CacheOptions{CleanupInterval: &zero},
			assertions: func(
				t *testing.T,
				_ *CacheOptions,
				c Cache[string, string],
				err error,
			) {
				require.ErrorContains(t, err, "CleanupInterval must be positive")
				require.Nil(t, c)
			},
		},
		{
			name: "negative CleanupInterval",
			opts: &CacheOptions{CleanupInterval: &negative},
			assertions: func(
				t *testing.T,
				_ *CacheOptions,
				c Cache[string, string],
				err error,
			) {
				require.ErrorContains(t, err, "CleanupInterval must be positive")
				require.Nil(t, c)
			},
		},
		{
			name: "CacheNothing",
			opts: &CacheOptions{CacheNothing: true},
			assertions: func(
				t *testing.T,
				_ *CacheOptions,
				c Cache[string, string],
				err error,
			) {
				require.NoError(t, err)
				require.Nil(t, internalValues(c))
			},
		},
		{
			// CleanupInterval governs eviction from a cache that CacheNothing keeps
			// from being created at all, so no value for it can be wrong here.
			name: "zero CleanupInterval with CacheNothing",
			opts: &CacheOptions{CleanupInterval: &zero, CacheNothing: true},
			assertions: func(
				t *testing.T,
				_ *CacheOptions,
				c Cache[string, string],
				err error,
			) {
				require.NoError(t, err)
				require.Nil(t, internalValues(c))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			c, err := NewCache(noopLoader, testCase.opts)
			testCase.assertions(t, testCase.opts, c, err)
		})
	}
}

func TestCache_Get_cacheHit(t *testing.T) {
	t.Parallel()

	// A retained value is served to subsequent callers without loading again.
	var runs atomic.Int32
	c := newTestCache(t, countingLoader(&runs, new(time.Hour)), nil)

	for range 2 {
		value, err := c.Get(t.Context(), testKey, testInput)
		require.NoError(t, err)
		require.Equal(t, testValue, value)
	}
	require.Equal(t, int32(1), runs.Load())
}

func TestCache_Get_uncachedValue(t *testing.T) {
	t.Parallel()

	// A Loader function returning a nil TTL leaves nothing for the next caller to
	// be served, so every caller loads.
	var runs atomic.Int32
	c := newTestCache(t, countingLoader(&runs, nil), nil)

	for range 2 {
		value, err := c.Get(t.Context(), testKey, testInput)
		require.NoError(t, err)
		require.Equal(t, testValue, value)
	}
	require.Equal(t, int32(2), runs.Load())
	require.Empty(t, internalValues(c).Items())
}

func TestCache_Get_cacheNothing(t *testing.T) {
	t.Parallel()

	// A Cache that caches nothing loads for every caller, whatever TTL its Loader
	// function returns.
	var runs atomic.Int32
	c := newTestCache(
		t,
		countingLoader(&runs, new(time.Hour)),
		&CacheOptions{CacheNothing: true},
	)

	for range 2 {
		value, err := c.Get(t.Context(), testKey, testInput)
		require.NoError(t, err)
		require.Equal(t, testValue, value)
	}
	require.Equal(t, int32(2), runs.Load())
}

func TestCache_Get_loadError(t *testing.T) {
	t.Parallel()

	// A failed load is reported to the caller and caches nothing, so the next
	// caller tries again. The TTL returned alongside the error is deliberately
	// one that would otherwise have been honored.
	var runs atomic.Int32
	c := newTestCache(
		t,
		func(context.Context, string) (string, *time.Duration, error) {
			runs.Add(1)
			return testValue, new(time.Hour), errors.New("something went wrong")
		},
		nil,
	)

	for range 2 {
		value, err := c.Get(t.Context(), testKey, testInput)
		require.ErrorContains(t, err, "something went wrong")
		require.Empty(t, value)
	}
	require.Equal(t, int32(2), runs.Load())
	require.Empty(t, internalValues(c).Items())
}

func TestCache_Get_loaderPanics(t *testing.T) {
	t.Parallel()

	// A panic escaping a Loader function into singleflight would be re-raised on a
	// goroutine of its own and take the process down with it. Every caller waiting
	// on that load is told about it as an error instead, and this test's own
	// survival is the evidence that the process was not lost.
	var runs atomic.Int32
	c := newTestCache(
		t,
		func(context.Context, string) (string, *time.Duration, error) {
			runs.Add(1)
			panic("something went wrong")
		},
		nil,
	)

	// A panicking load caches nothing, so a subsequent caller tries again.
	for range 2 {
		value, err := c.Get(t.Context(), testKey, testInput)
		require.ErrorContains(t, err, "something went wrong")
		require.Empty(t, value)
	}
	require.Equal(t, int32(2), runs.Load())
	require.Empty(t, internalValues(c).Items())
}

func TestCache_Get_nilValueOfAnInterfaceType(t *testing.T) {
	t.Parallel()

	// A value is wrapped on its way into the internal cache so that recovering it
	// again does not depend on its own type. Stored bare, a nil value of an
	// interface type comes back as a nil any, which no assertion to that type can
	// succeed against, so every caller would load anew.
	var runs atomic.Int32
	c, err := NewCache(
		func(context.Context, string) (any, *time.Duration, error) {
			runs.Add(1)
			return nil, new(time.Hour), nil
		},
		nil,
	)
	require.NoError(t, err)

	for range 2 {
		value, err := c.Get(t.Context(), testKey, testInput)
		require.NoError(t, err)
		require.Nil(t, value)
	}
	require.Equal(t, int32(1), runs.Load())
}

func TestCache_Get_ttl(t *testing.T) {
	t.Parallel()

	zero := time.Duration(0)
	negative := -1 * time.Minute
	tenMinutes := 10 * time.Minute
	twoHours := 2 * time.Hour

	testCases := []struct {
		name string
		// ttl is what the Loader function returns.
		ttl *time.Duration
		// optsDefaultTTL is what the Cache is configured with.
		optsDefaultTTL *time.Duration
		assertions     func(t *testing.T, item gocache.Item, retained bool)
	}{
		{
			name: "nil TTL leaves the value uncached",
			assertions: func(t *testing.T, _ gocache.Item, retained bool) {
				require.False(t, retained)
			},
		},
		{
			name: "positive TTL is used as given",
			ttl:  &tenMinutes,
			assertions: func(t *testing.T, item gocache.Item, retained bool) {
				require.True(t, retained)
				requireExpiresIn(t, item, tenMinutes)
			},
		},
		{
			// go-cache records an entry that never expires as one whose expiration
			// is zero.
			name: "negative TTL retains the value indefinitely",
			ttl:  &negative,
			assertions: func(t *testing.T, item gocache.Item, retained bool) {
				require.True(t, retained)
				require.Zero(t, item.Expiration)
			},
		},
		{
			name:           "zero TTL defers to DefaultTTL",
			ttl:            &zero,
			optsDefaultTTL: &twoHours,
			assertions: func(t *testing.T, item gocache.Item, retained bool) {
				require.True(t, retained)
				requireExpiresIn(t, item, twoHours)
			},
		},
		{
			name: "zero TTL defers to the default DefaultTTL",
			ttl:  &zero,
			assertions: func(t *testing.T, item gocache.Item, retained bool) {
				require.True(t, retained)
				requireExpiresIn(t, item, defaultTTL)
			},
		},
		{
			name:           "zero TTL defers to a DefaultTTL that is not positive",
			ttl:            &zero,
			optsDefaultTTL: &zero,
			assertions: func(t *testing.T, item gocache.Item, retained bool) {
				require.True(t, retained)
				require.Zero(t, item.Expiration)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			c := newTestCache(
				t,
				func(context.Context, string) (string, *time.Duration, error) {
					return testValue, testCase.ttl, nil
				},
				&CacheOptions{DefaultTTL: testCase.optsDefaultTTL},
			)

			value, err := c.Get(t.Context(), testKey, testInput)
			require.NoError(t, err)
			require.Equal(t, testValue, value)

			item, retained := internalValues(c).Items()[testKey]
			testCase.assertions(t, item, retained)
		})
	}
}

func TestCache_Get_coalescing(t *testing.T) {
	t.Parallel()

	const concurrency = 10

	// Callers are split evenly between two keys. Each key must be loaded exactly
	// once, which makes this sensitive both to callers failing to share a load
	// and to callers for different keys sharing one.
	keys := []string{"key-a", "key-b"}

	// synctest.Test() runs a function in a "bubble": that function and every
	// goroutine it starts form a group the runtime tracks. Inside the bubble,
	// synctest.Wait() calls block until every other goroutine in the group is
	// durably blocked -- parked on something only another member of the group can
	// release. That is what lets this test observe the moment when every caller
	// is waiting on a load, rather than guess at it with a sleep.
	synctest.Test(t, func(t *testing.T) {
		var runs atomic.Int32

		// This channel will be used to unblock the goroutines that are parked
		// inside the Loader function, but only after every caller is durably
		// blocked on a singleflight.
		release := make(chan struct{})

		// A Cache that caches nothing leaves loading as the only way any caller can
		// obtain a value. It also runs no janitor, a goroutine that would never
		// exit, and a bubble does not close until every goroutine within it has.
		c := newTestCache(
			t,
			func(_ context.Context, input string) (string, *time.Duration, error) {
				runs.Add(1)
				<-release
				// The value identifies the input it was loaded from, so a caller served
				// by another key's load is detectable.
				return "value-for-" + input, nil, nil
			},
			&CacheOptions{CacheNothing: true},
		)

		values := make([]string, concurrency)
		errs := make([]error, concurrency)
		var wg sync.WaitGroup
		for i := range concurrency {
			wg.Go(func() {
				key := keys[i%len(keys)]
				values[i], errs[i] = c.Get(t.Context(), key, key)
			})
		}

		// Returns only once every caller is durably blocked. The Cache retains
		// nothing, so the only place they can be is waiting on a load.
		// Specifically, singleflight has started one goroutine per key and each is
		// parked inside the Loader function, with every caller waiting on the load
		// it joined.
		synctest.Wait()

		// Allows the loads to complete.
		close(release)

		// Wait for all of the callers to finish.
		wg.Wait()

		// Since we forced the callers for each key to join one singleflight, there
		// should have been exactly one load per key -- no more, which would mean
		// callers failed to coalesce, and no fewer, which would mean callers for
		// different keys shared a load.
		require.Equal(t, int32(len(keys)), runs.Load())

		// Verify that every caller received the value for its own key and no
		// errors.
		for i := range concurrency {
			require.NoError(t, errs[i])
			require.Equal(t, "value-for-"+keys[i%len(keys)], values[i])
		}
	})
}

func TestCache_Get_coalescingIsKeyedOnTheKey(t *testing.T) {
	t.Parallel()

	const concurrency = 4

	// A key is what identifies work to be shared, so concurrent callers
	// presenting one key join a single load even where the inputs they present
	// differ. Every other test pairs one key with one input, which cannot tell
	// coalescing by key apart from coalescing by input.
	synctest.Test(t, func(t *testing.T) {
		var runs atomic.Int32

		release := make(chan struct{})
		c := newTestCache(
			t,
			func(_ context.Context, input string) (string, *time.Duration, error) {
				runs.Add(1)
				<-release
				return "value-for-" + input, nil, nil
			},
			&CacheOptions{CacheNothing: true},
		)

		var wg sync.WaitGroup
		for i := range concurrency {
			wg.Go(func() {
				_, _ = c.Get(t.Context(), testKey, fmt.Sprintf("%s-%d", testInput, i))
			})
		}

		// Returns once every caller is durably blocked on the load it joined.
		synctest.Wait()
		close(release)
		wg.Wait()

		require.Equal(t, int32(1), runs.Load())
	})
}

func TestCache_Get_winnerCanceled(t *testing.T) {
	t.Parallel()

	// We will test the scenario where the caller whose call began a load abandons
	// it mid-flight. Because a load runs under a context detached from any
	// caller's, it must survive that and still serve the remaining callers
	// waiting on it.

	const numWaiters = 3

	// The phased launch below relies on synctest.Wait(), which returns only once
	// every other goroutine in the bubble is durably blocked. That is what lets
	// this test know with certainty which caller began the load and which merely
	// joined it, and then know that all of them are waiting before the winner
	// walks away.
	synctest.Test(t, func(t *testing.T) {
		// This channel will be used to unblock the goroutine that is parked inside
		// the Loader function, but only after the winner has abandoned it.
		release := make(chan struct{})

		c := newTestCache(
			t,
			func(ctx context.Context, _ string) (string, *time.Duration, error) {
				// Honoring the context is what gives this test its teeth. Were the load
				// running under the winner's context instead of a detached one,
				// canceling the winner would land in the first case below and every
				// waiter would receive that error.
				select {
				case <-ctx.Done():
					return "", nil, ctx.Err()
				case <-release:
					return testValue, nil, nil
				}
			},
			&CacheOptions{CacheNothing: true},
		)

		// The winner is launched alone, and with a context only it holds.
		winnerCtx, cancel := context.WithCancel(t.Context())
		winnerErr := make(chan error, 1)
		go func() {
			_, err := c.Get(winnerCtx, testKey, testInput)
			winnerErr <- err
		}()

		// Returns once the winner and the goroutine singleflight started on its
		// behalf are both durably blocked: the latter parked inside the Loader
		// function, the former waiting on it. The load therefore exists and belongs
		// to the winner. Launching the waiters any earlier would leave ownership of
		// the load to the scheduler, and canceling a caller that does not own it
		// proves nothing.
		synctest.Wait()

		type result struct {
			value string
			err   error
		}
		waiterRes := make(chan result, numWaiters)
		for range numWaiters {
			go func() {
				value, err := c.Get(t.Context(), testKey, testInput)
				waiterRes <- result{value: value, err: err}
			}()
		}

		// Returns once the waiters have joined the winner's load, leaving the
		// goroutine singleflight started parked inside the Loader function and
		// every caller waiting on that load.
		synctest.Wait()

		// The winner abandons the load. The waiters' contexts stay live, so they
		// keep waiting.
		cancel()

		// The winner reports that it stopped waiting, rather than reporting a
		// failed load.
		winErr := <-winnerErr
		require.ErrorIs(t, winErr, context.Canceled)
		require.ErrorContains(t, winErr, "loading will continue in the background")

		// Because the load's context is detached, the winner's cancellation should
		// not have halted it. Closing the release channel unblocks the load. Doing
		// so before collecting the waiters' results below matters. If every
		// goroutine in the bubble were parked, the fake clock would advance to the
		// load's own deadline and fail it for reasons having nothing to do with
		// what is under test.
		close(release)

		// Verify that every waiter received the loaded value and no errors, meaning
		// the load outlived the caller that started it.
		for range numWaiters {
			res := <-waiterRes
			require.NoError(t, res.err)
			require.Equal(t, testValue, res.value)
		}
	})
}

func TestCache_Get_loadTimeout(t *testing.T) {
	t.Parallel()

	// A load runs under a context detached from every caller's, so a caller
	// giving up cannot end it. Its own deadline is the only thing that can.

	// A bubble has a fake clock that advances only when every goroutine in the
	// group is durably blocked, and then only as far as the next pending timer.
	// Here that timer is the load's own deadline, so this test reaches it
	// instantly and measures it exactly, instead of waiting thirty seconds for
	// it.
	synctest.Test(t, func(t *testing.T) {
		loadTimeout := testLoadTimeout
		c := newTestCache(t, hangingLoader, &CacheOptions{
			LoadTimeout:  &loadTimeout,
			CacheNothing: true,
		})

		start := time.Now()
		value, err := c.Get(t.Context(), testKey, testInput)

		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Empty(t, value)
		require.Equal(t, testLoadTimeout, time.Since(start))
	})
}

func TestCache_Get_defaultLoadTimeout(t *testing.T) {
	t.Parallel()

	// CacheOptions naming no LoadTimeout leave a load bounded by
	// defaultLoadTimeout rather than unbounded.
	synctest.Test(t, func(t *testing.T) {
		c := newTestCache(t, hangingLoader, &CacheOptions{CacheNothing: true})

		start := time.Now()
		value, err := c.Get(t.Context(), testKey, testInput)

		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Empty(t, value)
		require.Equal(t, defaultLoadTimeout, time.Since(start))
	})
}

// noopLoader suits tests concerned with a Cache's construction rather than with
// anything it loads.
func noopLoader(context.Context, string) (string, *time.Duration, error) {
	return testValue, nil, nil
}

// countingLoader returns a Loader function that records how many times it has
// run and returns testValue with the given TTL.
func countingLoader(
	runs *atomic.Int32,
	ttl *time.Duration,
) Loader[string, string] {
	return func(context.Context, string) (string, *time.Duration, error) {
		runs.Add(1)
		return testValue, ttl, nil
	}
}

// hangingLoader never returns on its own, so only its context ending can end it.
func hangingLoader(
	ctx context.Context,
	_ string,
) (string, *time.Duration, error) {
	<-ctx.Done()
	return "", nil, ctx.Err()
}

// newTestCache returns a Cache built from the given Loader function and
// CacheOptions, failing the test if those CacheOptions are rejected.
func newTestCache(
	t *testing.T,
	loader Loader[string, string],
	opts *CacheOptions,
) Cache[string, string] {
	t.Helper()
	c, err := NewCache(loader, opts)
	require.NoError(t, err)
	return c
}

// loadTimeoutOf returns the timeout a Cache will bound its loads by.
func loadTimeoutOf(c Cache[string, string]) time.Duration {
	return c.(*cache[string, string]).loadTimeout // nolint: forcetypeassert
}

// internalValues returns a Cache's internal cache, which tests concerned with
// what was retained, and for how long, inspect directly.
func internalValues(c Cache[string, string]) *gocache.Cache {
	return c.(*cache[string, string]).values // nolint: forcetypeassert
}

// requireExpiresIn asserts that a retained entry expires approximately the
// given duration from now.
func requireExpiresIn(t *testing.T, item gocache.Item, ttl time.Duration) {
	t.Helper()
	require.InDelta(
		t,
		ttl.Seconds(),
		time.Until(time.Unix(0, item.Expiration)).Seconds(),
		5,
	)
}
