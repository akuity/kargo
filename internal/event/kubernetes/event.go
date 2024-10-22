package kubernetes

import (
	"context"
	"errors"

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

type recorder struct {
	backoff wait.Backoff
	sink    record.EventSink
	logger  *logging.Logger

	newEventHandlerFn func(event *corev1.Event) func() error
}

// NewRecorder returns a new record.EventRecorder that records all events
// without aggregation, even the given event is correlated.
//
// NOTE: This recorder must be used with caution as it creates a new Event
// on every event without throttling / spam filtering features - which are
// included in the event recorder from k8s.io/client-go. This could lead
// to performance issues with the Kubernetes API server.
func NewRecorder(
	ctx context.Context,
	scheme *runtime.Scheme,
	client libClient.Client,
	name string,
) record.EventRecorder {
	logger := logging.LoggerFromContext(ctx)
	internalRecorder := newRecorder(ctx, client, logger)
	b := record.NewBroadcaster()
	b.StartEventWatcher(internalRecorder.handleEvent)
	return b.NewRecorder(
		scheme,
		corev1.EventSource{
			Component: name,
		},
	)
}

func newRecorder(
	ctx context.Context,
	client libClient.Client,
	logger *logging.Logger,
) *recorder {
	r := &recorder{
		backoff: retry.DefaultRetry, // TODO: Make it configurable
		sink:    newSink(ctx, client),
		logger:  logger,
	}
	r.newEventHandlerFn = r.createEvent
	return r
}

func (r *recorder) handleEvent(event *corev1.Event) {
	if err := retry.OnError(
		r.backoff,
		r.newRetryDecider(event),
		r.newEventHandlerFn(event),
	); err != nil {
		r.logger.Error(
			err, "Unable to handle event",
			"event", event,
		)
	}
}

func (r *recorder) createEvent(event *corev1.Event) func() error {
	return func() error {
		// Always create event instead of patching correlated events
		_, err := r.sink.Create(event)
		return err
	}
}

// newRetryDecider returns a function that decides whether
// to re-record event or not on given error.
func (r *recorder) newRetryDecider(event *corev1.Event) func(error) bool {
	return func(err error) bool {
		logger := r.logger.WithValues("event", event)

		var statusErr *apierrors.StatusError
		if errors.As(err, &statusErr) {
			if apierrors.IsAlreadyExists(err) ||
				apierrors.HasStatusCause(err, corev1.NamespaceTerminatingCause) {
				logger.Info(
					"Server rejected event (will not retry!)",
					"error", err,
				)
				return false
			}
			// Retry on other status errors
		}
		logger.Error(err, "Unable to write event (may retry after backoff)")
		return true
	}
}

var (
	_ record.EventSink = &sink{}
)

type sink struct {
	ctx    context.Context
	client libClient.Client
}

func newSink(
	ctx context.Context,
	client libClient.Client,
) *sink {
	return &sink{
		ctx:    ctx,
		client: client,
	}
}

func (e *sink) Create(event *corev1.Event) (*corev1.Event, error) {
	err := e.client.Create(e.ctx, event)
	return event, err
}

func (e *sink) Update(event *corev1.Event) (*corev1.Event, error) {
	err := e.client.Update(e.ctx, event)
	return event, err
}

func (e *sink) Patch(event *corev1.Event, data []byte) (*corev1.Event, error) {
	err := e.client.Patch(e.ctx, event, libClient.RawPatch(types.StrategicMergePatchType, data))
	return event, err
}
