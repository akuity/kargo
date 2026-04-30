package event

import (
	"context"
)

type Sender interface {
	// Send sends the event to the configured destination, returning an error if the send fails.
	Send(ctx context.Context, evt Meta) error

	// NOTE(thomastaylor312): We've never done any handling of event sending shutdown before,
	// and it was out of scope when I added this in. But now it is available if we want to be
	// more graceful about draining events when the controller shuts down

	// Shutdown drains any buffered events, blocking until they have been
	// delivered or the underlying transport gives up. Callers should
	// invoke Shutdown during graceful termination to avoid losing queued
	// events.
	Shutdown()
}
