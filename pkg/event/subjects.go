package event

import (
	"fmt"
	"strings"
)

const (
	// BasePrefix is the base subject prefix used for all Kargo events, including those published by
	// system components. It is the prefix that should be used in NATS subscriptions to receive all
	// Kargo events.
	BasePrefix = "akuity.kargo"

	// EventsSubjectPrefix is the subject prefix used for the user-facing events published by Kargo
	// components. Other system components will publish events on other subjects
	EventsSubjectPrefix = BasePrefix + ".events"

	// DefaultKind is the default kind subject segment used for user-facing events published by
	// Kargo components when the event is not related to a specific object kind.
	DefaultKind = "GLOBAL"
)

// NewKargoSubject returns the given topic prefixed with the global Kargo subject prefix.
//
// For example, if the topic is "my-topic", the returned subject will be "akuity.kargo.my-topic".
func NewKargoSubject(topic string) string {
	return fmt.Sprintf("%s.%s", BasePrefix, topic)
}

// NOTE(thomastaylor312): As we continue to add other system events, we may want to move this and
// the other existing event types into a `user` subpackage

// NewEventsSubjectPrefix constructs the subject for a user-facing event published by a Kargo
// component. The subject is of the form "akuity.kargo.events.<kind>", where <kind> is
// the provided event kind (which object it is related to) in lowercase, and GLOBAL otherwise.
// DefaultKind stays uppercase so that it can never collide with a lowercased kind.
func NewEventsSubjectPrefix(kind string) string {
	if kind == "" {
		return fmt.Sprintf("%s.%s", EventsSubjectPrefix, DefaultKind)
	}
	return fmt.Sprintf("%s.%s", EventsSubjectPrefix, strings.ToLower(kind))
}
