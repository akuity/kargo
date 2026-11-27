package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

// SetSSEHeaders configures the standard headers for Server-Sent Events (SSE)
// streaming on a gin context.
func SetSSEHeaders(c *gin.Context) {
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

// errorFromWatchStartError maps startup failures from Kubernetes Watch calls
// into the same Connect errors used for watch.Error events.
func errorFromWatchStartError(err error) error {
	if err == nil {
		return nil
	}
	if !apierrors.IsResourceExpired(err) {
		return err
	}
	return connect.NewError(
		connect.CodeOutOfRange,
		fmt.Errorf("watch resource version expired: %s", err.Error()),
	)
}

// sendSSEWatchError sends a watch error as an SSE error event. Watch endpoints
// may have already sent response headers, so HTTP status codes are no longer a
// reliable way to report expired resource versions after streaming begins.
func sendSSEWatchError(c *gin.Context, err error) {
	logger := logging.LoggerFromContext(c.Request.Context())

	message := err.Error()
	if connectErr, ok := errors.AsType[*connect.Error](err); ok {
		message = connectErr.Message()
	}

	event := struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{
		Code:    connect.CodeOf(err).String(),
		Message: message,
	}
	data, marshalErr := json.Marshal(event)
	if marshalErr != nil {
		logger.Error(marshalErr, "failed to marshal watch error event")
		return
	}
	if _, writeErr := fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", data); writeErr != nil {
		logger.Debug("failed to write watch error event", "error", writeErr)
		return
	}
	c.Writer.Flush()
}

// SendSSEWatchStartError sends startup watch errors that are part of the watch
// protocol as SSE error events. It returns true when the error was handled.
func SendSSEWatchStartError(c *gin.Context, err error) bool {
	watchErr := errorFromWatchStartError(err)
	if connect.CodeOf(watchErr) != connect.CodeOutOfRange {
		return false
	}
	SetSSEHeaders(c)
	sendSSEWatchError(c, watchErr)
	return true
}

// convertAndSendWatchEventFiltered converts a watch event's object to the target type
// and sends it as an SSE event. It handles both unstructured objects (from real
// clients) and typed objects (from fake clients in tests). Returns true if the
// caller should continue processing (including on conversion errors), false if
// the watch should be terminated (write failure or a watch.Error event).
// Only sends events for objects for which filter evaluates to "true", see filteredWatchEventType
func convertAndSendWatchEventFiltered[T any, Event any](
	c *gin.Context,
	watcher storageEventsWatcher[T, Event],
	e Event,
	filter func(T) bool,
) bool {
	if err := watcher.ErrorFromEvent(e); err != nil {
		sendSSEWatchError(c, err)
		return false
	}
	obj, ok := watcher.ObjectFromEvent(c.Request.Context(), e)
	if !ok {
		return true // continue processing, don't terminate watch
	}

	eventType := watcher.EventType(e)
	var send bool
	if filter != nil {
		// If filter is "true", eventType is unchanged
		eventType, send = filteredWatchEventType(eventType, filter(obj))
		if !send {
			return true
		}
	}
	return sendSSEWatchEvent(c, eventType, obj)
}

// filteredWatchEventType returns the event type to send for a client-side
// filtered watch event. Kubernetes server-side selectors send a DELETED event
// when a previously matching object is modified so it no longer matches; this
// helper mirrors that behavior for filters we must evaluate in-process.
//
// Because we hold no per-client matched set, we cannot tell whether a
// non-matching MODIFIED object previously matched, so we emit a synthetic
// DELETED for every non-matching MODIFIED event — including objects the client
// never received an ADDED for. This over-emits DELETEs relative to a real
// server-side selector, but a DELETE for an object the client is not tracking
// is a harmless no-op. Achieving exact fidelity would require tracking sent
// object identities per client, which is not worth the complexity here.
func filteredWatchEventType(eventType watch.EventType, matches bool) (watch.EventType, bool) {
	if matches {
		return eventType, true
	}
	if eventType == watch.Modified {
		return watch.Deleted, true
	}
	return "", false
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

// WriteSSEKeepalive writes a keepalive comment to keep the SSE connection
// alive. Returns true if successful, false if the write failed.
func WriteSSEKeepalive(c *gin.Context) bool {
	if _, err := c.Writer.Write([]byte(": keepalive\n\n")); err != nil {
		logger := logging.LoggerFromContext(c.Request.Context())
		logger.Debug("failed to write keepalive", "error", err)
		return false
	}
	c.Writer.Flush()
	return true
}

type storageEventsWatcher[T any, Event any] interface {
	Stop()
	ResultChan() <-chan Event

	ErrorFromEvent(Event) error
	ObjectFromEvent(context.Context, Event) (T, bool)
	EventType(Event) watch.EventType
}

type storageEventsWatcherK8s[T any] struct {
	watcher watch.Interface
}

func (s storageEventsWatcherK8s[T]) Stop() {
	s.watcher.Stop()
}
func (s storageEventsWatcherK8s[T]) ResultChan() <-chan watch.Event {
	return s.watcher.ResultChan()
}

// ErrorFromEvent maps Kubernetes watch.Error events into client-visible
// errors that watch handlers can return or stream.
func (s storageEventsWatcherK8s[T]) ErrorFromEvent(e watch.Event) error {
	if e.Type != watch.Error {
		return nil
	}

	status, ok := e.Object.(*metav1.Status)
	if !ok {
		return connect.NewError(
			connect.CodeUnknown,
			fmt.Errorf("watch error: unexpected object type %T", e.Object),
		)
	}

	message := status.Message
	if message == "" {
		message = string(status.Reason)
	}
	if status.Code == http.StatusGone || status.Reason == metav1.StatusReasonExpired {
		return connect.NewError(
			connect.CodeOutOfRange,
			fmt.Errorf("watch resource version expired: %s", message),
		)
	}
	return connect.NewError(
		connect.CodeUnknown,
		fmt.Errorf("watch error: %s", message),
	)
}

// ObjectFromEvent converts a watch event's object to the target type.
// It handles both unstructured objects (from real clients) and typed objects
// (from fake clients in tests). Returns the converted object and true if
// successful, or the zero value and false if conversion failed.
func (s storageEventsWatcherK8s[T]) ObjectFromEvent(ctx context.Context, e watch.Event) (T, bool) {
	logger := logging.LoggerFromContext(ctx)

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

func (s storageEventsWatcherK8s[T]) EventType(e watch.Event) watch.EventType {
	return e.Type
}

// Assertion that K8s watcher implements the interface
var _ storageEventsWatcher[any, watch.Event] = (*storageEventsWatcherK8s[any])(nil)

func (s *server) watchFreightEvents(
	ctx context.Context,
	project string,
	resourceVersion string,
) (storageEventsWatcher[*kargoapi.Freight, watch.Event], error) {
	return s.watchK8sResources[*kargoapi.Freight](
		ctx,
		&kargoapi.FreightList{},
		buildWatchListOptions(project, resourceVersion)...,
	)
}

// TODO: review that when switching types to DB
func (s *server) watchK8sResources[T any](
	ctx context.Context,
	list client.ObjectList,
	listOpts ...client.ListOption,
) (storageEventsWatcher[T, watch.Event], error) {
	w, err := s.client.Watch(
		ctx,
		list,
		listOpts...,
	)
	if err != nil {
		return nil, err
	}
	return storageEventsWatcherK8s[T]{watcher: w}, nil
}

func (s *server) watchResources[T any, Event any](
	c *gin.Context,
	watcher storageEventsWatcher[T, Event],
	filter func(T) bool,
) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	defer watcher.Stop()

	keepaliveTicker := time.NewTicker(30 * time.Second)
	defer keepaliveTicker.Stop()

	SetSSEHeaders(c)

	for {
		select {
		case <-ctx.Done():
			logger.Debug("watch context done", "error", ctx.Err())
			return

		case <-keepaliveTicker.C:
			if !WriteSSEKeepalive(c) {
				return
			}

		case e, ok := <-watcher.ResultChan():
			if !ok {
				logger.Debug("watch channel closed")
				return
			}

			if !convertAndSendWatchEventFiltered(c, watcher, e, filter) {
				return
			}
		}
	}
}
