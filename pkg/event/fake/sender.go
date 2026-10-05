// Package fake provides a fake event.Sender for use in tests.
package fake

import (
	"context"
	"sync"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
)

// Sender is an event.Sender that records every event sent to it instead of
// delivering it anywhere. It is safe for concurrent use.
type Sender struct {
	// SendErr, if non-nil, is returned by Send, and the event is not recorded.
	SendErr error

	mu   sync.Mutex
	sent []cloudevents.Event
}

// Send records the given event with its subject set as a real Sender would set
// it, "<subjectPrefix>.<event type>", or returns SendErr if it is non-nil.
func (s *Sender) Send(
	_ context.Context,
	subjectPrefix string,
	evt cloudevents.Event,
) error {
	if s.SendErr != nil {
		return s.SendErr
	}
	evt.SetSubject(subjectPrefix + "." + evt.Type())
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = append(s.sent, evt)
	return nil
}

// Sent returns every event recorded so far, in the order they were sent.
func (s *Sender) Sent() []cloudevents.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	sent := make([]cloudevents.Event, len(s.sent))
	copy(sent, s.sent)
	return sent
}

// Shutdown is a no-op.
func (s *Sender) Shutdown() {}
