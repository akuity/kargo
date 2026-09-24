package dbreconcile

// Predicate decides which events are worth reconciling, as controller-runtime
// predicates do. Each function judges events of one kind and reports whether
// to let them through. A nil function lets every event of its kind through,
// so a Predicate need only set the functions it cares about.
//
// Predicates only ever judge events. Keys a controller's resync enumerates
// are not events and are never filtered.
type Predicate[K comparable, T any] struct {
	Create func(Event[K, T]) bool
	Update func(Event[K, T]) bool
	Delete func(Event[K, T]) bool
}

// Allow reports whether the predicate lets the event through. An event of an
// unknown kind is never let through.
func (p Predicate[K, T]) Allow(e Event[K, T]) bool {
	var allow func(Event[K, T]) bool
	switch e.Kind {
	case Created:
		allow = p.Create
	case Updated:
		allow = p.Update
	case Deleted:
		allow = p.Delete
	default:
		return false
	}
	return allow == nil || allow(e)
}
