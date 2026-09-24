package dbreconcile

import "context"

// Lister enumerates the keys a controller's resync enqueues: every resource
// that may need work.
type Lister[K comparable] interface {
	// ListKeys returns up to limit keys that follow after, in a stable order.
	// The first page is requested with the zero value of K. A page shorter
	// than limit ends the enumeration. A query ordered by the key column, with
	// a condition that the key is greater than after, satisfies this.
	ListKeys(ctx context.Context, after K, limit int) ([]K, error)
}

// ListerFunc is a Lister implemented by a function.
type ListerFunc[K comparable] func(ctx context.Context, after K, limit int) ([]K, error)

// ListKeys implements Lister.
func (f ListerFunc[K]) ListKeys(ctx context.Context, after K, limit int) ([]K, error) {
	return f(ctx, after, limit)
}
