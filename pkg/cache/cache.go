package cache

import "context"

// Cache defines a simple interface for cache operations.
type Cache[V any] interface {
	// Get retrieves a value from the cache. If found, returns the value, true,
	// and a nil error. If not found, returns a zero value, false, and a nil
	// error. In the event of an error, returns a zero value, false, and the
	// error. Callers should always inspect the last two return values before
	// trusting the first.
	Get(ctx context.Context, key string) (V, bool, error)
	// Set stores a value in the cache. In the event of an error, the error is
	// returned.
	Set(ctx context.Context, key string, value V) error
}
