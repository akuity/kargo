package event

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	libClient "sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libargocd "github.com/akuity/kargo/internal/argocd"
	"github.com/akuity/kargo/internal/directives"
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

// NewPromotionEventAnnotations returns annotations for a Promotion related event.
// It may skip some fields when error occurred during serialization, to record event with best-effort.
func NewPromotionEventAnnotations(
	ctx context.Context,
	actor string,
	p *kargoapi.Promotion,
	f *kargoapi.Freight,
) map[string]string {
	logger := logging.LoggerFromContext(ctx)

	annotations := map[string]string{
		kargoapi.AnnotationKeyEventProject:             p.GetNamespace(),
		kargoapi.AnnotationKeyEventPromotionName:       p.GetName(),
		kargoapi.AnnotationKeyEventFreightName:         p.Spec.Freight,
		kargoapi.AnnotationKeyEventStageName:           p.Spec.Stage,
		kargoapi.AnnotationKeyEventPromotionCreateTime: p.GetCreationTimestamp().Format(time.RFC3339),
	}

	if actor != "" {
		annotations[kargoapi.AnnotationKeyEventActor] = actor
	}
	// All Promotion-related events are emitted after the promotion was created.
	// Therefore, if the promotion knows who triggered it, set them as an actor.
	if promoteActor, ok := p.Annotations[kargoapi.AnnotationKeyCreateActor]; ok {
		annotations[kargoapi.AnnotationKeyEventActor] = promoteActor
	}

	if f != nil {
		annotations[kargoapi.AnnotationKeyEventFreightCreateTime] = f.CreationTimestamp.Format(time.RFC3339)
		annotations[kargoapi.AnnotationKeyEventFreightAlias] = f.Alias
		if len(f.Commits) > 0 {
			data, err := json.Marshal(f.Commits)
			if err != nil {
				logger.Error(err, "marshal freight commits in JSON")
			} else {
				annotations[kargoapi.AnnotationKeyEventFreightCommits] = string(data)
			}
		}
		if len(f.Images) > 0 {
			data, err := json.Marshal(f.Images)
			if err != nil {
				logger.Error(err, "marshal freight images in JSON")
			} else {
				annotations[kargoapi.AnnotationKeyEventFreightImages] = string(data)
			}
		}
		if len(f.Charts) > 0 {
			data, err := json.Marshal(f.Charts)
			if err != nil {
				logger.Error(err, "marshal freight charts in JSON")
			} else {
				annotations[kargoapi.AnnotationKeyEventFreightCharts] = string(data)
			}
		}
	}

	var apps []types.NamespacedName
	for _, step := range p.Spec.Steps {
		if step.Uses != "argocd-update" || step.Config == nil {
			continue
		}
		var cfg directives.ArgoCDUpdateConfig
		if err := json.Unmarshal(step.Config.Raw, &cfg); err != nil {
			logger.Error(err, "unmarshal ArgoCD update config")
			continue
		}
		for _, app := range cfg.Apps {
			namespacedName := types.NamespacedName{
				Namespace: app.Namespace,
				Name:      app.Name,
			}
			if namespacedName.Namespace == "" {
				namespacedName.Namespace = libargocd.Namespace()
			}
			apps = append(apps, namespacedName)
		}
	}
	if len(apps) > 0 {
		data, err := json.Marshal(apps)
		if err != nil {
			logger.Error(err, "marshal ArgoCD apps in JSON")
		} else {
			annotations[kargoapi.AnnotationKeyEventApplications] = string(data)
		}
	}

	return annotations
}
