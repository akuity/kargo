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
// rather than per sender, which is why this doesn't use the CloudEvents SDK's
// NATS protocol binding.
type eventSender struct {
	client *nats.Conn
}

// NewDefaultingEventSender is a convenience wrapper around NewEventSender that automatically wraps
// it in a DefaultingSender
func NewDefaultingEventSender(client *nats.Conn, component string) event.Sender {
	return event.NewDefaultingSender(NewEventSender(client), component)
}

// NewEventSender creates a new event.Sender that publishes events as-is using
// the provided NATS client. Use NewDefaultingEventSender to also have each
// event's ID, time, and source set. On shutdown the sender will _flush_ the
// client, but not close the connection. The caller is responsible for closing
// the connection when it is no longer needed.
func NewEventSender(client *nats.Conn) event.Sender {
	return &eventSender{
		client: client,
	}
}

// Send publishes the event on the subject "<subjectPrefix>.<event type>",
// recording that subject as the event's subject attribute.
func (s *eventSender) Send(
	_ context.Context,
	subjectPrefix string,
	evt cloudevents.Event,
) error {
	// The subject has to be set before marshaling so that it is part of the
	// published event.
	evt.SetSubject(subjectPrefix + "." + evt.Type())
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
