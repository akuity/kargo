package kubeclient

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	libClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/logging"
)

type eventRecorder struct {
	backoff wait.Backoff
	sink    record.EventSink
	logger  *log.Entry

	newEventHandlerFn func(event *corev1.Event) func() error
}

// NewEventRecorder returns a new event recorder that records all events
// without aggregation, even the given event is correlated.
//
// NOTE: This recorder must be used with caution as it creates a new Event
// on every event without throttling / spam filtering features - which are
// included in the event recorder from k8s.io/client-go. This could lead
// to performance issues with the Kubernetes API server.
func NewEventRecorder(
	ctx context.Context,
	scheme *runtime.Scheme,
	client libClient.Client,
	name string,
) record.EventRecorder {
	logger := logging.LoggerFromContext(ctx)
	internalRecorder := newEventRecorder(ctx, client, logger)
	b := record.NewBroadcaster()
	b.StartEventWatcher(internalRecorder.handleEvent)
	return b.NewRecorder(
		scheme,
		corev1.EventSource{
			Component: name,
		},
	)
}

func newEventRecorder(
	ctx context.Context,
	client libClient.Client,
	logger *log.Entry,
) *eventRecorder {
	r := &eventRecorder{
		backoff: retry.DefaultRetry, // TODO: Make it configurable
		sink:    newEventSink(ctx, client),
		logger:  logger,
	}
	r.newEventHandlerFn = r.createEvent
	return r
}

func (r *eventRecorder) handleEvent(event *corev1.Event) {
	if err := retry.OnError(
		r.backoff,
		r.newRetryDecider(event),
		r.newEventHandlerFn(event),
	); err != nil {
		r.logger.WithError(err).Error("Unable to handle event", "event", event)
	}
}

func (r *eventRecorder) createEvent(event *corev1.Event) func() error {
	return func() error {
		// Always create event instead of patching correlated events
		_, err := r.sink.Create(event)
		return err
	}
}

// newRetryDecider returns a function that decides whether
// to re-record event or not on given error.
func (r *eventRecorder) newRetryDecider(event *corev1.Event) func(error) bool {
	return func(err error) bool {
		logger := r.logger.
			WithField("event", event).
			WithError(err)

		var statusErr *apierrors.StatusError
		if errors.As(err, &statusErr) {
			if apierrors.IsAlreadyExists(err) ||
				apierrors.HasStatusCause(err, corev1.NamespaceTerminatingCause) {
				logger.Info("Server rejected event (will not retry!)")
				return false
			}
			// Retry on other status errors
		}
		logger.Error("Unable to write event (may retry after backoff)")
		return true
	}
}

var (
	_ record.EventSink = &eventSink{}
)

type eventSink struct {
	ctx    context.Context
	client libClient.Client
}

func newEventSink(
	ctx context.Context,
	client libClient.Client,
) *eventSink {
	return &eventSink{
		ctx:    ctx,
		client: client,
	}
}

func (e *eventSink) Create(event *corev1.Event) (*corev1.Event, error) {
	err := e.client.Create(e.ctx, event)
	return event, err
}

func (e *eventSink) Update(event *corev1.Event) (*corev1.Event, error) {
	err := e.client.Update(e.ctx, event)
	return event, err
}

func (e *eventSink) Patch(event *corev1.Event, data []byte) (*corev1.Event, error) {
	err := e.client.Patch(e.ctx, event, libClient.RawPatch(types.StrategicMergePatchType, data))
	return event, err
}
