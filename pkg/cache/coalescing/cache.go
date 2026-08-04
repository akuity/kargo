package coalescing

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"golang.org/x/sync/singleflight"

	"github.com/akuity/kargo/pkg/logging"
)

const (
	// defaultCleanupInterval is how often a Cache evicts expired entries when no
	// CacheOptions.CleanupInterval is provided. Eviction reclaims only what no
	// caller asks for again, so this is not urgent work, but a coarse interval
	// would retain such entries for many multiples of a short TTL. The cost of a
	// finer one grows with the number of entries, and a Cache holding enough of
	// them for that to matter is one whose CleanupInterval was chosen
	// deliberately.
	defaultCleanupInterval = time.Minute
	// defaultLoadTimeout is the default timeout for a Loader function when no
	// CacheOptions.LoadTimeout is provided.
	defaultLoadTimeout = time.Minute
	// defaultTTL is how long a Cache retains a value whose Loader function
	// deferred that decision, when no CacheOptions.DefaultTTL is provided.
	defaultTTL = time.Hour
)

// Loader is the signature for any function that, given implementation-specific
// input, returns a value for a Cache to cache, along with the TTL to cache it
// for, and if applicable, an error. A nil TTL signals that the value is not to
// be cached. TTL of zero implicitly defers to associated
// CacheOptions.DefaultTTL. A negative duration signals that the value is to be
// cached indefinitely. Implementations are required to honor the context
// argument and return promptly when it is canceled, timed out, or otherwise
// ended.
type Loader[I, T any] func(
	ctx context.Context,
	input I,
) (T, *time.Duration, error)

// CacheOptions are passed to NewCache() to configure the returned Cache.
// NewCache() will accept a nil CacheOptions pointer and create its own
// CacheOptions instead. Whether CacheOptions are passed in or created,
// NewCache() works from a copy, populating any nil fields of that copy with
// sensible defaults. A Cache's configuration is therefore fixed at the moment
// it is built and unaffected by any later change to the CacheOptions the caller
// retains.
type CacheOptions struct {
	// CleanupInterval is how often to evict expired entries from the Cache. When
	// nil, NewCache() substitutes a default of 1m. A zero or negative duration is
	// invalid and will cause NewCache() to return an error. This field is ignored
	// entirely when CacheNothing is true, there being nothing to evict.
	CleanupInterval *time.Duration
	// DefaultTTL specifies how long to cache values obtained from a Loader
	// function when the TTL also returned from that Loader function is zero. When
	// nil, NewCache() substitutes a default of 1h. A zero or negative duration is
	// valid and indicates the effective DefaultTTL is indefinite.
	DefaultTTL *time.Duration
	// LoadTimeout bounds execution of a Loader function. When nil, NewCache()
	// substitutes a default of 1m. A zero or negative duration is invalid and
	// will cause NewCache() to return an error.
	LoadTimeout *time.Duration
	// CacheNothing, when true, effectively disables all internal caching. When
	// true, DefaultTTL and CleanupInterval have no effect, regardless of their
	// values. This option is useful primarily for tests that wish to
	// deterministically avoid all caching and ensure every call to Cache.Get()
	// executes a Loader function or waits on one already in-flight.
	CacheNothing bool
}

// Cache is an interface for a load-through cache.
type Cache[I, T any] interface {
	// Get returns the value cached under the given key, if one exists. On a miss,
	// it returns the value a Loader function obtains using the given input, and
	// caches that value unless the Loader function or this Cache's own
	// configuration directs otherwise. Crucially, implementations MUST coalesce
	// concurrent loads for a single key such that potentially expensive work is
	// done only once, with all applicable, concurrent callers receiving the same
	// result when loading completes. Subsequent callers presenting that key MUST
	// receive the cached value, where one was cached, until it expires.
	//
	// A key and an input are separate arguments because the caller often will
	// have derived the key from the input using a transformation (like a hash,
	// for instance) that cannot be reversed, yet the original input will
	// typically be required by an underlying Loader function in the event of a
	// cache miss.
	//
	// A caller whose own context ends before a load completes stops waiting and
	// receives an error saying so. The load itself carries on, belonging as it
	// does to no one caller, so that those still waiting on it are served.
	Get(ctx context.Context, key string, input I) (T, error)
}

// entry wraps a value on its way into and out of a cache that holds values of
// any type, so that recovering the value again does not depend on the value's
// own type. Stored bare, a nil value of an interface type T comes back as a nil
// any, which no assertion to T can succeed against, so the value would be
// retained and yet never served and every caller would load anew.
type entry[T any] struct {
	value T
}

// cache is an implementation of the Cache interface.
type cache[I, T any] struct {
	// loader is the function that will be called to load a value when a cache
	// miss occurs.
	loader Loader[I, T]

	// loadTimeout bounds execution of loader.
	loadTimeout time.Duration

	// values is this component's INTERNAL cache. When nil, it stands down with
	// all calls to Get() executing a Loader function or waiting on one already
	// in-flight.
	values *gocache.Cache

	// group coalesces concurrent loads for any given key into a single load whose
	// result is shared by all callers.
	group singleflight.Group
}

// NewCache returns a Cache that fills misses using the given Loader function.
// It returns an error if the given CacheOptions are invalid.
func NewCache[I, T any](
	loader Loader[I, T],
	opts *CacheOptions,
) (Cache[I, T], error) {
	// Defaults are applied to a copy, leaving the caller's CacheOptions as they
	// were found. The durations they name are dereferenced below, so nothing the
	// caller still holds a pointer to can alter this Cache after the fact.
	var effectiveOpts CacheOptions
	if opts != nil {
		effectiveOpts = *opts
	}
	if effectiveOpts.CleanupInterval == nil {
		effectiveOpts.CleanupInterval = new(defaultCleanupInterval)
	}
	if effectiveOpts.DefaultTTL == nil {
		effectiveOpts.DefaultTTL = new(defaultTTL)
	}
	if effectiveOpts.LoadTimeout == nil {
		effectiveOpts.LoadTimeout = new(defaultLoadTimeout)
	}
	if *effectiveOpts.LoadTimeout <= 0 {
		return nil,
			errors.New("invalid CacheOptions: LoadTimeout must be positive")
	}
	c := &cache[I, T]{
		loader:      loader,
		loadTimeout: *effectiveOpts.LoadTimeout,
	}
	if !effectiveOpts.CacheNothing {
		if *effectiveOpts.CleanupInterval <= 0 {
			return nil,
				errors.New("invalid CacheOptions: CleanupInterval must be positive")
		}
		c.values = gocache.New(
			*effectiveOpts.DefaultTTL,
			*effectiveOpts.CleanupInterval,
		)
	}
	return c, nil
}

// Get implements the Cache interface.
func (c *cache[I, T]) Get(
	ctx context.Context,
	key string,
	input I,
) (T, error) {
	var zero T

	logger := logging.LoggerFromContext(ctx).WithValues("key", key)

	if c.values != nil {
		if cached, exists := c.values.Get(key); exists {
			if e, ok := cached.(entry[T]); ok {
				logger.Debug("cache hit")
				return e.value, nil
			}
		}
		logger.Debug("cache miss")
	} else {
		logger.Debug("internal cache disabled; skipping cache lookup")
	}

	ch := c.group.DoChan(key, func() (val any, err error) {
		// A panic escaping into singleflight is re-raised on a goroutine of its
		// own, where no recovery a caller has in place can reach it, and takes the
		// process down. Reporting it as an error instead keeps a defective Loader
		// function from doing that, at the cost of the panic's own stack trace,
		// which is logged here since it will not appear anywhere else.
		defer func() {
			if r := recover(); r != nil {
				logger.Error(
					nil, "recovered from panic in Loader function",
					"panic", r,
					"stack", string(debug.Stack()),
				)
				err = fmt.Errorf("Loader function panicked: %v", r)
			}
		}()
		return c.loadAndCache(ctx, key, input)
	})

	select {
	case <-ctx.Done():
		return zero, fmt.Errorf(
			"load interrupted; loading will continue in the background: %w",
			ctx.Err(),
		)
	case res := <-ch:
		if res.Err != nil {
			return zero, res.Err
		}
		loaded, _ := res.Val.(entry[T])
		return loaded.value, nil
	}
}

// loadAndCache detaches the load from the calling context, runs it, and caches
// what it produced, if that value is to be cached. It is intended to be
// executed within the Cache's singleflight group.
func (c *cache[I, T]) loadAndCache(
	ctx context.Context,
	key string,
	input I,
) (entry[T], error) {
	logger := logging.LoggerFromContext(ctx).WithValues("key", key)

	// Since no caller's cancellation bounds this work, it carries its own
	// timeout. Nothing else of the calling context survives but its logger,
	// carried over so that what the load logs remains attributable to the call
	// that began it.
	orphanedCtx, cancel := context.WithTimeout(
		logging.ContextWithLogger(context.Background(), logger),
		c.loadTimeout,
	)
	defer cancel()

	value, ttl, err := c.loader(orphanedCtx, input)
	if err != nil {
		return entry[T]{}, err
	}
	loaded := entry[T]{value: value}
	if c.values != nil && ttl != nil {
		c.values.Set(key, loaded, *ttl)
	}
	return loaded, nil
}
