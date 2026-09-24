package dbreconcile

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
)

// EventKind says what happened to a resource.
type EventKind string

const (
	// Created means the resource was created. The event carries it as New.
	Created EventKind = "Created"
	// Updated means the resource changed. The event carries it as it was, as
	// Old, and as it is now, as New.
	Updated EventKind = "Updated"
	// Deleted means the resource was deleted. The event carries it as it was
	// last, as Old.
	Deleted EventKind = "Deleted"
)

// Event describes a change to one resource of type T, identified by a key of
// type K. It is the body of every message on a subject, encoded as JSON:
//
//	{"kind":"Updated","key":"7b1c...","old":{...},"new":{...}}
//
// Which of Old and New are present is determined by Kind; Publish and Subject
// both reject an event that lacks the ones its kind requires, so predicates
// may rely on them.
type Event[K comparable, T any] struct {
	Kind EventKind `json:"kind"`
	Key  K         `json:"key"`
	Old  *T        `json:"old,omitempty"`
	New  *T        `json:"new,omitempty"`
}

// validate reports whether the event carries what its kind requires.
func (e Event[K, T]) validate() error {
	switch e.Kind {
	case Created:
		if e.New == nil {
			return errors.New("a Created event requires the new resource")
		}
	case Updated:
		if e.Old == nil || e.New == nil {
			return errors.New("an Updated event requires both the old and the new resource")
		}
	case Deleted:
		if e.Old == nil {
			return errors.New("a Deleted event requires the old resource")
		}
	default:
		return fmt.Errorf("unknown event kind %q", e.Kind)
	}
	return nil
}

// Publish publishes an event to a subject. Callers publish only once the
// write the event describes has committed; an event for a write that is
// later rolled back would announce something that never happened.
//
// Publish is a no-op when conn is nil, so that components which may run
// without NATS need no special casing. A lost event is not an error either:
// a controller's resync picks up whatever its events missed.
func Publish[K comparable, T any](conn *nats.Conn, subject string, e Event[K, T]) error {
	if conn == nil {
		return nil
	}
	if subject == "" {
		return errors.New("subject is required")
	}
	if err := e.validate(); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("error encoding event: %w", err)
	}
	if err = conn.Publish(subject, data); err != nil {
		return fmt.Errorf("error publishing event to %q: %w", subject, err)
	}
	return nil
}
