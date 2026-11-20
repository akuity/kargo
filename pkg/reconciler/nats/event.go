package nats

import (
	"context"
	"errors"
	"fmt"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"

	"github.com/akuity/kargo/pkg/event"
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
// type K. On the wire it is a CloudEvent whose type is the topic's resource
// followed by the kind, such as "PromotionRolloutUpdated", and whose data is
// the rest of the event, encoded as JSON:
//
//	{"key":{"Project":"p","Name":"t"},"old":{...},"new":{...}}
//
// T is the database's representation of the resource, the type its store
// reads and writes, so that an event says exactly what was committed and
// filters can compare rows without a conversion.
//
// Which of Old and New are present is determined by Kind. Publishing and
// subscribing both reject an event that lacks what its kind requires, so
// filters and mappers may rely on them.
type Event[K comparable, T any] struct {
	// Kind travels as the CloudEvent's type rather than in its data.
	Kind EventKind `json:"-"`
	Key  K         `json:"key"`
	Old  *T        `json:"old,omitempty"`
	New  *T        `json:"new,omitempty"`
}

// Validate reports whether the event carries what its kind requires.
func (e Event[K, T]) Validate() error {
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

// InternalSubjectPrefix roots the subject tree of every topic. Topics carry
// internal events, which tell controllers what was written, so they travel
// apart from the user-facing events published under akuity.kargo.events:
// the two serve different purposes, and a subscriber to one never receives
// the other, even when both are published for the same write.
const InternalSubjectPrefix = "akuity.platform.internal"

// Topic names the kind of resource a set of events is about and fixes, once,
// the types of their keys and rows. Declare one beside the store that owns
// the resource and use it from both the publisher and the controllers, so
// neither spells out the types again:
//
//	var TargetEvents = nats.NewTopic[reconciler.Key, database.TargetRow]("Target")
//
// Events are CloudEvents of the types <resource>Created, <resource>Updated
// and <resource>Deleted, published to the subjects event.SubjectFromType
// derives from those types under InternalSubjectPrefix: a "PromotionRollout"
// topic publishes to akuity.platform.internal.promotion.rollout.created, and
// so on. A subscription to the topic receives all three, and nothing
// published for another resource, even one whose subjects share a prefix
// with the topic's, such as a "Promotion" topic's.
type Topic[K comparable, T any] struct {
	resource string
	// subjectPrefix is the subject an event's kind is appended to.
	subjectPrefix string
}

// NewTopic returns the topic for the named resource, such as
// "PromotionRollout". The name must be a valid event type, which
// event.SubjectSuffix describes; NewTopic panics if it is not, since a topic
// is declared once, from a literal, when its package is initialized.
func NewTopic[K comparable, T any](resource string) Topic[K, T] {
	subjectPrefix, err := event.SubjectFromType(InternalSubjectPrefix, resource)
	if err != nil {
		panic(fmt.Sprintf("invalid topic resource %q: %v", resource, err))
	}
	return Topic[K, T]{resource: resource, subjectPrefix: subjectPrefix}
}

// Subject returns the wildcard subject that matches every event on the
// topic. It matches a single token after the topic's subject prefix, the
// event's kind, so that it does not also match the subjects of a resource
// whose name extends this one's.
func (t Topic[K, T]) Subject() string { return t.subjectPrefix + ".*" }

// eventType returns the CloudEvent type of events of one kind.
func (t Topic[K, T]) eventType(kind EventKind) string {
	return t.resource + string(kind)
}

// Decode returns the event a CloudEvent published to the topic carries. It
// returns an error if the CloudEvent is not one of the topic's types, or
// carries less than its kind requires.
func (t Topic[K, T]) Decode(evt cloudevents.Event) (Event[K, T], error) {
	kind, ok := strings.CutPrefix(evt.Type(), t.resource)
	if !ok {
		return Event[K, T]{}, fmt.Errorf("event type %q is not one of topic %s's", evt.Type(), t.resource)
	}
	var e Event[K, T]
	if err := evt.DataAs(&e); err != nil {
		return Event[K, T]{}, fmt.Errorf("error decoding %s event: %w", evt.Type(), err)
	}
	e.Kind = EventKind(kind)
	if err := e.Validate(); err != nil {
		return Event[K, T]{}, fmt.Errorf("invalid %s event: %w", evt.Type(), err)
	}
	return e, nil
}

// Mapper turns an event into the requests a controller should reconcile.
// The request type is the controller's, not the event's: a Stage controller
// maps a PromotionRequest event to the Stage named in the row.
//
// A Mapper runs on the subscription's delivery goroutine, so it should be
// quick; a slow one delays every later event on the same topic.
type Mapper[K comparable, T any, request any] func(context.Context, Event[K, T]) []request
