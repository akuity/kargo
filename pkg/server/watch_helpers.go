package server

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/akuity/kargo/pkg/logging"
)

// setSSEHeaders configures the standard headers for Server-Sent Events (SSE)
// streaming on a gin context.
func setSSEHeaders(c *gin.Context) {
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

// convertWatchEventObject converts a watch event's object to the target type.
// It handles both unstructured objects (from real clients) and typed objects
// (from fake clients in tests). Returns the converted object and true if
// successful, or the zero value and false if conversion failed.
func convertWatchEventObject[T any](c *gin.Context, e watch.Event, _ T) (T, bool) {
	logger := logging.LoggerFromContext(c.Request.Context())

	var obj T
	switch o := e.Object.(type) {
	case *unstructured.Unstructured:
		// Handle unstructured objects (from real client)
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(o.Object, &obj); err != nil {
			logger.Error(err, "failed to convert from unstructured")
			return obj, false
		}
	default:
		// Handle typed objects (from fake client in tests)
		typed, ok := e.Object.(T)
		if !ok {
			logger.Error(
				fmt.Errorf("unexpected object type: %T", e.Object),
				"failed to convert watch event object",
			)
			return obj, false
		}
		obj = typed
	}
	return obj, true
}

// convertAndSendWatchEvent converts a watch event's object to the target type
// and sends it as an SSE event. It handles both unstructured objects (from real
// clients) and typed objects (from fake clients in tests). Returns true if the
// caller should continue processing (including on conversion errors), false if
// the watch should be terminated (write failure).
func convertAndSendWatchEvent[T any](c *gin.Context, e watch.Event, target T) bool {
	obj, ok := convertWatchEventObject(c, e, target)
	if !ok {
		return true // continue processing, don't terminate watch
	}
	return sendSSEWatchEvent(c, e.Type, obj)
}

// sendSSEWatchEvent sends an object as an SSE watch event. Returns true if
// the caller should continue processing, false if the watch should be
// terminated (write failure).
func sendSSEWatchEvent[T any](c *gin.Context, eventType watch.EventType, obj T) bool {
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

// writeSSEKeepalive writes a keepalive comment to keep the SSE connection
// alive. Returns true if successful, false if the write failed.
func writeSSEKeepalive(c *gin.Context) bool {
	if _, err := c.Writer.Write([]byte(": keepalive\n\n")); err != nil {
		logger := logging.LoggerFromContext(c.Request.Context())
		logger.Debug("failed to write keepalive", "error", err)
		return false
	}
	c.Writer.Flush()
	return true
}
