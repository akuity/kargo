package nats

import (
	"context"
	"encoding/json"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/nats-io/nats.go"

	"github.com/akuity/kargo/pkg/event"
)

// contentTypeHeader is the NATS message header that carries the content type
// of a structured-mode CloudEvent.
const contentTypeHeader = "Content-Type"

// eventSender is an implementation of event.Sender that publishes CloudEvents
// to NATS in structured content mode, i.e. the whole event, including its
// attributes, is the JSON message payload. The subject is chosen per event
// (under a per-sender prefix) rather than per sender, which is why this doesn't
// use the CloudEvents SDK's NATS protocol binding.
type eventSender struct {
	client        *nats.Conn
	subjectPrefix string
}

// NewDefaultingEventSender is a convenience wrapper around NewEventSender that automatically wraps
// it in a DefaultingSender
func NewDefaultingEventSender(
	client *nats.Conn,
	subjectPrefix string,
	component string,
) event.Sender {
	return event.NewDefaultingSender(
		NewEventSender(client, subjectPrefix),
		component,
	)
}

// NewEventSender creates a new event.Sender that publishes events as-is using
// the provided NATS client, under the given subject prefix (e.g.
// event.EventsSubjectPrefix). Use NewDefaultingEventSender to also have each
// event's ID, time, and source set. On shutdown the sender will _flush_ the
// client, but not close the connection. The caller is responsible for closing
// the connection when it is no longer needed.
func NewEventSender(client *nats.Conn, subjectPrefix string) event.Sender {
	return &eventSender{
		client:        client,
		subjectPrefix: subjectPrefix,
	}
}

// Send publishes the event on the subject returned by event.SubjectFromType
// for the sender's prefix and the event's type, recording that subject as the
// event's subject attribute.
func (s *eventSender) Send(_ context.Context, evt cloudevents.Event) error {
	subject, err := event.SubjectFromType(s.subjectPrefix, evt.Type())
	if err != nil {
		return fmt.Errorf("error building subject for CloudEvent: %w", err)
	}
	// The subject has to be set before marshaling so that it is part of the
	// published event.
	evt.SetSubject(subject)
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("error marshaling CloudEvent: %w", err)
	}
	msg := nats.NewMsg(evt.Subject())
	msg.Header.Set(contentTypeHeader, cloudevents.ApplicationCloudEventsJSON)
	msg.Data = data
	return s.client.PublishMsg(msg)
}

// Shutdown flushes the NATS client, blocking until all buffered events have been sent or the
// underlying transport gives up. Closing of the NATS connection is left to the caller, so that it
// can be shared with other components and closed at the appropriate time.
func (s *eventSender) Shutdown() {
	s.client.Flush()
}
