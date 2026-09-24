package server

import (
	"context"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/akuity/kargo/pkg/logging"
)

// polledWatch describes how to observe a set of database-backed objects for
// an SSE watch. The database offers no change feed, so the set is polled and
// the differences between successive snapshots are streamed as watch events.
type polledWatch[T any] struct {
	// snapshot returns the objects as they are now, already filtered to the
	// ones the watch is for.
	snapshot func(context.Context) ([]T, error)
	// name identifies an object across snapshots.
	name func(T) string
	// version changes whenever an object changes.
	version func(T) string
}

// servePolledWatch streams changes to a set of database-backed objects as SSE
// watch events, in the same envelope the Kubernetes-backed watch endpoints
// use. The first snapshot is replayed as ADDED events unless seed carries a
// resource version, in which case only objects newer than it are sent, as
// MODIFIED events; a client that lists first and then watches from the list's
// resource version thus sees everything that changed in between and nothing
// it already has. Every interval thereafter a new snapshot is taken and its
// differences from the last are sent: new objects as ADDED, changed ones as
// MODIFIED and vanished ones as DELETED.
func servePolledWatch[T any](
	c *gin.Context,
	interval time.Duration,
	seed string,
	source polledWatch[T],
) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	items, err := source.snapshot(ctx)
	if err != nil {
		_ = c.Error(err)
		return
	}

	SetSSEHeaders(c)

	known := make(map[string]T, len(items))
	seedVersion, seeded := parseResourceVersion(seed)
	for _, item := range items {
		known[source.name(item)] = item
		eventType := watch.Added
		if seeded {
			version, ok := parseResourceVersion(source.version(item))
			if ok && version <= seedVersion {
				continue
			}
			eventType = watch.Modified
		}
		if !SendSSEWatchEvent(c, eventType, item) {
			return
		}
	}

	pollTicker := time.NewTicker(interval)
	defer pollTicker.Stop()
	keepaliveTicker := time.NewTicker(30 * time.Second)
	defer keepaliveTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Debug("watch context done", "error", ctx.Err())
			return

		case <-keepaliveTicker.C:
			if !WriteSSEKeepalive(c) {
				return
			}

		case <-pollTicker.C:
			items, err = source.snapshot(ctx)
			if err != nil {
				if ctx.Err() == nil {
					logger.Error(err, "failed to poll for watch")
					SendSSEWatchError(c, err)
				}
				return
			}
			var ok bool
			if known, ok = sendPolledChanges(c, source, known, items); !ok {
				return
			}
		}
	}
}

// sendPolledChanges sends the differences between the objects known to have
// been sent and the objects in the latest snapshot, and returns the latter
// keyed by name. It returns false when the stream can no longer be written.
func sendPolledChanges[T any](
	c *gin.Context,
	source polledWatch[T],
	known map[string]T,
	items []T,
) (map[string]T, bool) {
	current := make(map[string]T, len(items))
	for _, item := range items {
		name := source.name(item)
		current[name] = item
		previous, seen := known[name]
		switch {
		case !seen:
			if !SendSSEWatchEvent(c, watch.Added, item) {
				return current, false
			}
		case source.version(previous) != source.version(item):
			if !SendSSEWatchEvent(c, watch.Modified, item) {
				return current, false
			}
		}
	}
	// Vanished objects are reported in a stable order, carrying what was last
	// sent for them, as Kubernetes does.
	var gone []string
	for name := range known {
		if _, ok := current[name]; !ok {
			gone = append(gone, name)
		}
	}
	slices.Sort(gone)
	for _, name := range gone {
		if !SendSSEWatchEvent(c, watch.Deleted, known[name]) {
			return current, false
		}
	}
	return current, true
}
