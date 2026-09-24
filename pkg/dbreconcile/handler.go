package dbreconcile

import "context"

// EventHandler turns an event about a resource identified by a key of type
// SK into the keys, of type K, that a controller should reconcile.
type EventHandler[SK comparable, T any, K comparable] interface {
	Keys(ctx context.Context, e Event[SK, T]) []K
}

// MapFunc is an EventHandler implemented by a function.
type MapFunc[SK comparable, T any, K comparable] func(context.Context, Event[SK, T]) []K

// Keys implements EventHandler.
func (f MapFunc[SK, T, K]) Keys(ctx context.Context, e Event[SK, T]) []K {
	return f(ctx, e)
}

// EnqueueKey returns an EventHandler that reconciles the resource an event is
// about. It is what a controller uses for events about its own resource.
func EnqueueKey[K comparable, T any]() EventHandler[K, T, K] {
	return MapFunc[K, T, K](func(_ context.Context, e Event[K, T]) []K {
		return []K{e.Key}
	})
}

// EnqueueMapped returns an EventHandler that reconciles the resources a
// function maps an event to, as controller-runtime's
// EnqueueRequestsFromMapFunc does. It is what a controller uses for events
// about another resource: a child's event mapped to its parent's key, say.
//
// The function runs on the subscription's delivery goroutine, so it should be
// quick; a slow one delays every later event on the same subject.
func EnqueueMapped[SK comparable, T any, K comparable](
	fn func(context.Context, Event[SK, T]) []K,
) EventHandler[SK, T, K] {
	return MapFunc[SK, T, K](fn)
}
