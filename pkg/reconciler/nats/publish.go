package nats

import (
	"context"
	"errors"
	"fmt"

	natsgo "github.com/nats-io/nats.go"

	"github.com/akuity/kargo/pkg/event"
	natsevent "github.com/akuity/kargo/pkg/event/nats"
)

// Publisher publishes the events of every topic. It is a distinct type,
// rather than any event.Sender, so that the sender of user-facing events,
// which publishes under another prefix, cannot be handed to a store in its
// place.
//
// A nil *Publisher publishes nothing, so that components which may run
// without NATS need no special casing.
type Publisher struct {
	sender event.Sender
}

// NewPublisher returns a Publisher that publishes through conn, recording the
// named component, such as "api", as the source of its events.
func NewPublisher(conn *natsgo.Conn, component string) *Publisher {
	return NewPublisherWithSender(natsevent.NewDefaultingEventSender(
		conn,
		InternalSubjectPrefix,
		component,
	))
}

// NewPublisherWithSender returns a Publisher that publishes through the given
// sender, which must publish under InternalSubjectPrefix for subscribers to
// receive its events. Only use this function for tests or if you know what
// you're doing. Almost any other case should just use `NewPublisher`
func NewPublisherWithSender(sender event.Sender) *Publisher {
	return &Publisher{sender: sender}
}

// Shutdown flushes any events the publisher has buffered. It leaves the NATS
// connection open.
func (p *Publisher) Shutdown() {
	if p != nil {
		p.sender.Shutdown()
	}
}

// PublishCreated announces that the resource identified by key was created.
func (t Topic[K, T]) PublishCreated(
	ctx context.Context,
	pub *Publisher,
	key K,
	created T,
) error {
	return t.publish(ctx, pub, Event[K, T]{Kind: Created, Key: key, New: &created})
}

// PublishUpdated announces that the resource identified by key changed from
// old to updated.
func (t Topic[K, T]) PublishUpdated(
	ctx context.Context,
	pub *Publisher,
	key K,
	old, updated T,
) error {
	return t.publish(ctx, pub, Event[K, T]{Kind: Updated, Key: key, Old: &old, New: &updated})
}

// PublishDeleted announces that the resource identified by key, last seen
// as old, was deleted.
func (t Topic[K, T]) PublishDeleted(
	ctx context.Context,
	pub *Publisher,
	key K,
	old T,
) error {
	return t.publish(ctx, pub, Event[K, T]{Kind: Deleted, Key: key, Old: &old})
}

// publish publishes an event as a CloudEvent of the type for its kind.
// Callers publish only once the write the event describes has committed; an
// event for a write that is later rolled back would announce something that
// never happened.
//
// A failure to publish is not a failure of the write: the controller's
// resync picks up whatever its events missed, so callers log it and move on.
func (t Topic[K, T]) publish(ctx context.Context, pub *Publisher, e Event[K, T]) error {
	if pub == nil {
		return nil
	}
	if t.resource == "" {
		return errors.New("topic has no resource")
	}
	if err := e.Validate(); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}
	eventType := t.eventType(e.Kind)
	evt, err := event.NewCloudEvent(eventType, e)
	if err != nil {
		return fmt.Errorf("error creating %s event: %w", eventType, err)
	}
	evt.SetExtension(event.KindExtension, t.resource)
	if err = pub.sender.Send(ctx, evt); err != nil {
		return fmt.Errorf("error publishing %s event: %w", eventType, err)
	}
	return nil
}
