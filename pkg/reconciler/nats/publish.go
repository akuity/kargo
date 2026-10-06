package nats

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Publisher publishes a message to a subject. A *nats.Conn is one.
type Publisher interface {
	Publish(subject string, data []byte) error
}

// PublishCreated announces that the resource identified by key was created.
func (t Topic[K, T]) PublishCreated(pub Publisher, key K, created T) error {
	return t.publish(pub, Event[K, T]{Kind: Created, Key: key, New: &created})
}

// PublishUpdated announces that the resource identified by key changed from
// old to updated.
func (t Topic[K, T]) PublishUpdated(pub Publisher, key K, old, updated T) error {
	return t.publish(pub, Event[K, T]{Kind: Updated, Key: key, Old: &old, New: &updated})
}

// PublishDeleted announces that the resource identified by key, last seen
// as old, was deleted.
func (t Topic[K, T]) PublishDeleted(pub Publisher, key K, old T) error {
	return t.publish(pub, Event[K, T]{Kind: Deleted, Key: key, Old: &old})
}

// publish publishes an event to the subject for its kind. Callers publish
// only once the write the event describes has committed; an event for a
// write that is later rolled back would announce something that never
// happened.
//
// Publishing is a no-op when pub is nil, so that components which may run
// without NATS need no special casing. Callers hold a nil interface rather
// than an interface holding a nil *nats.Conn. A failure to publish is not a
// failure of the write either: the controller's resync picks up whatever its
// events missed, so callers log it and move on.
func (t Topic[K, T]) publish(pub Publisher, e Event[K, T]) error {
	if pub == nil {
		return nil
	}
	if t.prefix == "" {
		return errors.New("topic has no prefix")
	}
	if err := e.Validate(); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("error encoding event: %w", err)
	}
	subject := t.subjectFor(e.Kind)
	if err = pub.Publish(subject, data); err != nil {
		return fmt.Errorf("error publishing event to %q: %w", subject, err)
	}
	return nil
}
