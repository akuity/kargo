// Package natswatch serves watches of resources that live in the database
// rather than in Kubernetes. Their stores announce every committed write on
// a NATS topic, which stands in for the Kubernetes watch the API server uses
// for everything else. Clients see the same SSE stream either way.
package natswatch

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	natsgo "github.com/nats-io/nats.go"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/logging"
	natsrec "github.com/akuity/kargo/pkg/reconciler/nats"
	"github.com/akuity/kargo/pkg/server/sse"
)

// Serve streams one resource over SSE: the initial object, then MODIFIED
// or DELETED whenever an event on topic passes match, re-reading through
// load so it always sends current state. On each keepalive tick it also
// re-reads and sends MODIFIED if version changed, covering lost events.
// It unsubscribes when the request context ends.
//
// The initial object is sent as ADDED, as a Kubernetes watch sends an
// existing object. A re-read that finds the version already sent sends
// nothing, so an event the watch has already caught up with, or an event
// for a change that left the resource as it was, is not repeated. DELETED
// carries the resource as it was last sent, and ends the stream.
//
// load reports database.ErrNotFound, possibly wrapped, when the resource does
// not exist. If the first read fails, for that reason or any other, Serve
// records load's error on the context and returns before sending anything,
// so the error middleware answers with whatever status the error carries. A
// later read that fails for any other reason is logged and the watch goes
// on; the next event or tick reads again.
func Serve[K comparable, T any, O any](
	c *gin.Context,
	conn natsrec.Subscriber,
	topic natsrec.Topic[K, T],
	match func(natsrec.Event[K, T]) bool,
	load func(context.Context) (O, error),
	version func(O) string,
	keepalive time.Duration,
) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	// Subscribe before the first read, so that a change made between the two
	// is not missed. changed holds at most one pending signal: every re-read
	// returns the latest state, so further events before the next re-read
	// need no signal of their own.
	changed := make(chan struct{}, 1)
	sub, err := topic.Subscribe(conn, func(e natsrec.Event[K, T]) {
		if !match(e) {
			return
		}
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	if err != nil {
		_ = c.Error(fmt.Errorf("error watching: %w", err))
		return
	}
	defer func() {
		if unsubErr := sub.Unsubscribe(); unsubErr != nil &&
			!errors.Is(unsubErr, natsgo.ErrConnectionClosed) {
			logger.Error(unsubErr, "error unsubscribing")
		}
	}()

	current, err := load(ctx)
	if err != nil {
		_ = c.Error(err)
		return
	}
	sent := version(current)

	sse.SetHeaders(c)
	if !sse.SendWatchEvent(c, watch.Added, current) {
		return
	}

	ticker := time.NewTicker(keepalive)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !sse.WriteKeepalive(c) {
				return
			}
		case <-changed:
		}

		latest, err := load(ctx)
		if errors.Is(err, database.ErrNotFound) {
			sse.SendWatchEvent(c, watch.Deleted, current)
			return
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Error(err, "error re-reading watched resource")
			continue
		}
		if version(latest) == sent {
			continue
		}
		current, sent = latest, version(latest)
		if !sse.SendWatchEvent(c, watch.Modified, current) {
			return
		}
	}
}
