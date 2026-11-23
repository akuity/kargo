// Package sse writes the Server-Sent Events (SSE) stream every watch endpoint
// of the API server answers with, so that watches of Kubernetes resources
// and watches of resources that live in the database send the same wire
// format, the one pkg/x/client/watch reads. It depends on nothing else in
// the API server, so that any handler package can use it.
package sse

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/akuity/kargo/pkg/logging"
)

// SetHeaders configures the standard headers for an SSE stream on a gin
// context and sends them.
func SetHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable proxy buffering
	c.Writer.Flush()
}

// WatchEvent represents a watch event in an SSE stream. It contains the event
// type (ADDED, MODIFIED, DELETED) and the object that was affected.
type WatchEvent[T any] struct {
	Type   string `json:"type"`
	Object T      `json:"object"`
}

// SendWatchEvent sends an object as an SSE watch event. Returns true if the
// caller should continue processing, false if the watch should be terminated
// (write failure).
func SendWatchEvent[T any](c *gin.Context, eventType watch.EventType, obj T) bool {
	logger := logging.LoggerFromContext(c.Request.Context())

	event := WatchEvent[T]{
		Type:   string(eventType),
		Object: obj,
	}

	data, err := json.Marshal(event)
	if err != nil {
		logger.Error(err, "failed to marshal event")
		return true // continue processing, don't terminate watch
	}

	// Write as SSE event (format: data: <json>\n\n)
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
		logger.Debug("failed to write event", "error", err)
		return false // terminate watch
	}

	c.Writer.Flush()
	return true // continue processing
}

// WriteKeepalive writes a keepalive comment to keep the SSE connection alive.
// Returns true if successful, false if the write failed.
func WriteKeepalive(c *gin.Context) bool {
	if _, err := c.Writer.Write([]byte(": keepalive\n\n")); err != nil {
		logger := logging.LoggerFromContext(c.Request.Context())
		logger.Debug("failed to write keepalive", "error", err)
		return false
	}
	c.Writer.Flush()
	return true
}
