package event

import (
	"context"
)

type Sender interface {
	// Send sends the event to the configured destination, returning an error if the send fails.
	Send(ctx context.Context, evt Meta) error
}
