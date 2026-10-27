package event

import (
	"context"
	"time"
	"uuid"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
)

type Sender interface {
	// Send sends the CloudEvent to the configured destination, returning an error if the send
	// fails. Implementations may not use the provided context depending on the underlying
	// transport, but it is provided for completeness and future-proofing.
	//
	// Implementations should set the event's subject attribute to the same value, so that consumers
	// can filter on it. Each implementation can decide how it wants to construct _where_ it is
	// going to send the event (such as the subject to publish to in a pub/sub system). The
	// SubjectFromType(subjectPrefix, evt.Type()) method can be used to construct a subject from the
	// event type and a prefix, if desired.
	//
	// The passed event should not be used after this call, as it may be modified by the
	// implementation.
	Send(ctx context.Context, evt cloudevents.Event) error

	// Shutdown drains any buffered events, blocking until they have been delivered or the
	// underlying transport gives up. Callers should invoke Shutdown during graceful termination to
	// avoid losing queued events.
	Shutdown()
}

// DefaultingSender is a convenience wrapper around a Sender that automatically sets the UUID,
// timestamp, and source of events before sending them. It is useful for components that
// want to send events without having to manually set these attributes on every event. The
// underlying sender is exported so that it can be used directly if needed
type DefaultingSender struct {
	Sender

	source string
}

// NewDefaultingSender creates a new DefaultingSender that wraps the provided Sender and sets the
// source of events to the given component name (e.g. "api" or "stage-controller").
func NewDefaultingSender(sender Sender, component string) *DefaultingSender {
	return &DefaultingSender{
		Sender: sender,
		source: ComponentSource(component),
	}
}

func (s *DefaultingSender) Send(ctx context.Context, evt cloudevents.Event) error {
	// We use UUIDv7 here because it is time-ordered, which makes them sortable by creation time.
	evt.SetID(uuid.NewV7().String())
	evt.SetTime(time.Now())
	evt.SetSource(s.source)
	return s.Sender.Send(ctx, evt)
}
