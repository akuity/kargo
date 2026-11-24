package nats

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
// type K. It is the body of every message on a topic, encoded as JSON:
//
//	{"kind":"Updated","key":{"Project":"p","Name":"t"},"old":{...},"new":{...}}
//
// T is the database's representation of the resource, the type its store
// reads and writes, so that an event says exactly what was committed and
// filters can compare rows without a conversion.
//
// Which of Old and New are present is determined by Kind. Publishing and
// subscribing both reject an event that lacks what its kind requires, so
// filters and mappers may rely on them.
type Event[K comparable, T any] struct {
	Kind EventKind `json:"kind"`
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

// Topic names the subject tree events about one kind of resource travel on
// and fixes, once, the types of their keys and rows. Declare one beside the
// store that owns the resource and use it from both the publisher and the
// controllers, so neither spells out the types again:
//
//	var TargetEvents = nats.NewTopic[reconciler.Key, database.TargetRow]("kargo.targets")
//
// Events are published to <prefix>.created, <prefix>.updated and
// <prefix>.deleted, and a subscription to the topic receives all three.
type Topic[K comparable, T any] struct {
	prefix string
}

// NewTopic returns the topic rooted at prefix, such as "kargo.targets".
func NewTopic[K comparable, T any](prefix string) Topic[K, T] {
	return Topic[K, T]{prefix: prefix}
}

// Subject returns the wildcard subject that matches every event on the
// topic.
func (t Topic[K, T]) Subject() string { return t.prefix + ".>" }

// subjectFor returns the subject events of one kind are published to.
func (t Topic[K, T]) subjectFor(kind EventKind) string {
	return t.prefix + "." + strings.ToLower(string(kind))
}

// Mapper turns an event into the requests a controller should reconcile.
// The request type is the controller's, not the event's: a Stage controller
// maps a PromotionRequest event to the Stage named in the row.
//
// A Mapper runs on the subscription's delivery goroutine, so it should be
// quick; a slow one delays every later event on the same topic.
type Mapper[K comparable, T any, request any] func(context.Context, Event[K, T]) []request
