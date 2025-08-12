package event

import (
	"context"

	cloudevent "github.com/cloudevents/sdk-go/v2"
)

type Sender interface {
	// Send sends the event to the configured destination, returning an error if the send fails.
	Send(ctx context.Context, evt cloudevent.Event) error
}
