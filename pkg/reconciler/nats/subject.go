// Package nats is the NATS implementation of a reconciler.Source: a topic
// watched for events about resources. It also publishes those events, so
// that writers and watchers agree on the wire format.
//
// A Topic fixes the subject tree and the key and row types for one kind of
// resource, once. A publisher calls its Publish methods after each committed
// write, and a controller subscribes to it with Subject or Mapped. Neither
// spells out the types again.
//
// Core NATS delivers a message only to subscribers connected when it is
// published, so events can be lost: while a controller restarts, say. A
// controller watching a topic should therefore also list its store
// periodically. Moving to JetStream, or to an outbox fed from the
// database, changes this package and nothing above it.
package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	natsgo "github.com/nats-io/nats.go"

	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/reconciler"
)

// Subscriber subscribes to NATS subjects. A *nats.Conn is one.
type Subscriber interface {
	Subscribe(subject string, cb natsgo.MsgHandler) (*natsgo.Subscription, error)
	QueueSubscribe(subject, queue string, cb natsgo.MsgHandler) (*natsgo.Subscription, error)
}

// Source is a reconciler.Source fed by the events on a topic. Create one
// with Subject or Mapped, then narrow it with the On and Ignore methods.
type Source[K comparable, T any, request any] struct {
	conn       Subscriber
	topic      Topic[K, T]
	queueGroup string
	mapper     Mapper[K, T, request]
	onCreate   func(T) bool
	onUpdate   func(old, updated T) bool
	onDelete   func(T) bool
	ignored    map[EventKind]bool
}

var _ reconciler.Source[int] = (*Source[int, struct{}, int])(nil)

// Subject returns a source that pushes the key of the resource each event on
// the topic is about. It is what a controller keyed by that resource uses
// for events about it.
func Subject[K comparable, T any](conn Subscriber, topic Topic[K, T]) *Source[K, T, K] {
	return Mapped(conn, topic, func(_ context.Context, e Event[K, T]) []K {
		return []K{e.Key}
	})
}

// Mapped returns a source that pushes whatever the mapper derives from each
// event on the topic. It is what a controller uses for events about another
// resource: a Stage controller maps PromotionRequest events to Stages.
func Mapped[K comparable, T any, request any](
	conn Subscriber,
	topic Topic[K, T],
	mapper Mapper[K, T, request],
) *Source[K, T, request] {
	return &Source[K, T, request]{
		conn:    conn,
		topic:   topic,
		mapper:  mapper,
		ignored: map[EventKind]bool{},
	}
}

// OnCreate keeps only the Created events for which fn returns true.
func (s *Source[K, T, request]) OnCreate(fn func(created T) bool) *Source[K, T, request] {
	s.onCreate = fn
	return s
}

// OnUpdate keeps only the Updated events for which fn returns true. It is
// where a controller says which changes matter, so that its own status
// writes do not trigger the next reconcile.
func (s *Source[K, T, request]) OnUpdate(fn func(old, updated T) bool) *Source[K, T, request] {
	s.onUpdate = fn
	return s
}

// OnDelete keeps only the Deleted events for which fn returns true.
func (s *Source[K, T, request]) OnDelete(fn func(deleted T) bool) *Source[K, T, request] {
	s.onDelete = fn
	return s
}

// Ignore drops every event of the given kinds. A controller for a resource
// that needs no work once deleted ignores Deleted.
func (s *Source[K, T, request]) Ignore(kinds ...EventKind) *Source[K, T, request] {
	for _, kind := range kinds {
		s.ignored[kind] = true
	}
	return s
}

// QueueGroup makes the replicas of a controller share the topic's events,
// each event reaching one member of the named group, rather than every
// replica receiving every event. It is for controllers that run in every
// replica rather than only in the leader. Other controllers watching the
// same topic are in groups of their own, or in none, and still receive
// every event. The name must be unique per subscription: two sources of one
// controller sharing a group would each see only some of the events.
func (s *Source[K, T, request]) QueueGroup(name string) *Source[K, T, request] {
	s.queueGroup = name
	return s
}

// allow reports whether the event passes the source's filters.
func (s *Source[K, T, request]) allow(e Event[K, T]) bool {
	if s.ignored[e.Kind] {
		return false
	}
	switch e.Kind {
	case Created:
		return s.onCreate == nil || s.onCreate(*e.New)
	case Updated:
		return s.onUpdate == nil || s.onUpdate(*e.Old, *e.New)
	case Deleted:
		return s.onDelete == nil || s.onDelete(*e.Old)
	default:
		return false
	}
}

// Start implements reconciler.Source. It subscribes and returns; pushing
// stops when ctx is done. A message that is not a valid event is logged and
// dropped; the controller's resync recovers whatever it would have caused.
func (s *Source[K, T, request]) Start(ctx context.Context, out chan<- request) error {
	if s.conn == nil {
		return errors.New("a NATS connection is required")
	}
	if s.topic.prefix == "" {
		return errors.New("a topic is required")
	}
	if s.mapper == nil {
		return errors.New("a mapper is required")
	}
	subject := s.topic.Subject()
	logger := logging.LoggerFromContext(ctx).WithValues("subject", subject, "queueGroup", s.queueGroup)
	handle := func(msg *natsgo.Msg) {
		if ctx.Err() != nil {
			return
		}
		var e Event[K, T]
		if err := json.Unmarshal(msg.Data, &e); err != nil {
			logger.Error(err, "dropping message that is not an event", "messageSubject", msg.Subject)
			return
		}
		if err := e.Validate(); err != nil {
			logger.Error(err, "dropping invalid event", "messageSubject", msg.Subject)
			return
		}
		if !s.allow(e) {
			return
		}
		for _, req := range s.mapper(ctx, e) {
			select {
			case out <- req:
			case <-ctx.Done():
				return
			}
		}
	}
	var (
		sub *natsgo.Subscription
		err error
	)
	if s.queueGroup == "" {
		sub, err = s.conn.Subscribe(subject, handle)
	} else {
		sub, err = s.conn.QueueSubscribe(subject, s.queueGroup, handle)
	}
	if err != nil {
		return fmt.Errorf("error subscribing to %q: %w", subject, err)
	}
	go func() {
		<-ctx.Done()
		if err := sub.Unsubscribe(); err != nil && !errors.Is(err, natsgo.ErrConnectionClosed) {
			logger.Error(err, "error unsubscribing")
		}
	}()
	return nil
}
