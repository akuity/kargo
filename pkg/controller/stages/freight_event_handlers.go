package stages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/nats-io/nats.go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	kargoEvent "github.com/akuity/kargo/pkg/event"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/logging"
)

// These handlers are used to notify stage reconcilers about Freight events via NATS.
// This is a replacement of k8s watch on freight.

// freightEventChannelBufferSize bounds how many events may queue between a NATS
// subscription callback and the controller before the callback blocks. Core
// NATS delivery is best-effort, so a bounded buffer with the reconciler's
// periodic resync as the safety net is acceptable.
const freightEventChannelBufferSize = 1024

// subscribeFreightEvent subscribes to the NATS subject carrying the Kargo
// CloudEvents of the given type and returns a channel of GenericEvents -- one
// per event -- for a source.Channel watch to consume. For each message, decode
// maps the CloudEvent to the Freight to enqueue against: a non-nil error drops
// the message with a log, a nil Freight drops it silently (e.g. missing fields).
// The subscription is torn down when ctx is done.
func subscribeFreightEvent(
	ctx context.Context,
	conn *nats.Conn,
	eventType kargoapi.EventType,
	decode func(cloudevents.Event) (*kargoapi.Freight, error),
) (<-chan event.TypedGenericEvent[*kargoapi.Freight], error) {
	subject, err := kargoEvent.NewEventSubject(string(eventType))
	if err != nil {
		return nil, fmt.Errorf("error building subject for %s events: %w", eventType, err)
	}

	logger := logging.LoggerFromContext(ctx).WithValues("subject", subject)
	ch := make(chan event.TypedGenericEvent[*kargoapi.Freight], freightEventChannelBufferSize)

	sub, err := conn.Subscribe(subject, func(msg *nats.Msg) {
		if ctx.Err() != nil {
			return
		}
		var ce cloudevents.Event
		if unmarshalErr := json.Unmarshal(msg.Data, &ce); unmarshalErr != nil {
			logger.Error(unmarshalErr, "dropping message that is not a CloudEvent")
			return
		}
		freight, decodeErr := decode(ce)
		if decodeErr != nil {
			logger.Error(decodeErr, "dropping event with undecodable data")
			return
		}
		if freight == nil {
			return
		}
		select {
		case ch <- event.TypedGenericEvent[*kargoapi.Freight]{Object: freight}:
		case <-ctx.Done():
		}
	})
	if err != nil {
		return nil, fmt.Errorf("error subscribing to %q: %w", subject, err)
	}

	go func() {
		<-ctx.Done()
		if err := sub.Unsubscribe(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
			logger.Error(err, "error unsubscribing", "subject", subject)
		}
	}()

	return ch, nil
}

// subscribeFreightCreated subscribes to FreightCreated events. Each resulting
// GenericEvent carries a Freight whose Namespace and Origin.Name identify the
// Warehouse whose subscribed Stages should be reconciled.
func subscribeFreightCreated(
	ctx context.Context,
	conn *nats.Conn,
) (<-chan event.TypedGenericEvent[*kargoapi.Freight], error) {
	return subscribeFreightEvent(
		ctx,
		conn,
		kargoapi.EventTypeFreightCreated,
		func(ce cloudevents.Event) (*kargoapi.Freight, error) {
			var data kargoEvent.FreightCreated
			if err := ce.DataAs(&data); err != nil {
				return nil, err
			}
			if data.Project == "" || data.WarehouseName == "" {
				return nil, nil
			}
			return &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: data.Project,
					Name:      data.Name,
				},
				Origin: kargoapi.FreightOrigin{Name: data.WarehouseName},
			}, nil
		},
	)
}

// subscribeFreightVerifiedInStage subscribes to FreightVerifiedInStage events.
// Each resulting GenericEvent carries a Freight whose Namespace and
// Status.VerifiedIn identify the upstream Stage in which the Freight was verified,
// used to resolve the downstream Stages to reconcile.
func subscribeFreightVerifiedInStage(
	ctx context.Context,
	conn *nats.Conn,
) (<-chan event.TypedGenericEvent[*kargoapi.Freight], error) {
	return subscribeFreightEvent(
		ctx,
		conn,
		kargoapi.EventTypeFreightVerifiedInStage,
		func(ce cloudevents.Event) (*kargoapi.Freight, error) {
			var data kargoEvent.FreightVerifiedInStage
			if err := ce.DataAs(&data); err != nil {
				return nil, err
			}
			if data.Project == "" || data.StageName == "" {
				return nil, nil
			}
			return &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: data.Project,
					Name:      data.Name,
				},
				Origin: kargoapi.FreightOrigin{Name: data.WarehouseName},
				Status: kargoapi.FreightStatus{
					VerifiedIn: map[string]kargoapi.VerifiedStage{data.StageName: {}},
				},
			}, nil
		},
	)
}

// subscribeFreightCurrentlyInStage subscribes to FreightCurrentlyInStage events,
// emitted whenever a Freight becomes newly current in a Stage. Each resulting
// GenericEvent carries a Freight whose Namespace and Status.CurrentlyIn identify
// the upstream Stage it is now current in; the event's VerifiedInStage flag is
// encoded into Status.VerifiedIn so the enqueuer can apply the verification
// filter.
func subscribeFreightCurrentlyInStage(
	ctx context.Context,
	conn *nats.Conn,
) (<-chan event.TypedGenericEvent[*kargoapi.Freight], error) {
	return subscribeFreightEvent(
		ctx,
		conn,
		kargoapi.EventTypeFreightCurrentlyInStage,
		func(ce cloudevents.Event) (*kargoapi.Freight, error) {
			var data kargoEvent.FreightCurrentlyInStage
			if err := ce.DataAs(&data); err != nil {
				return nil, err
			}
			if data.Project == "" || data.StageName == "" {
				return nil, nil
			}
			status := kargoapi.FreightStatus{
				CurrentlyIn: map[string]kargoapi.CurrentStage{data.StageName: {}},
			}
			if data.VerifiedInStage {
				status.VerifiedIn = map[string]kargoapi.VerifiedStage{data.StageName: {}}
			}
			return &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: data.Project,
					Name:      data.Name,
				},
				Origin: kargoapi.FreightOrigin{Name: data.WarehouseName},
				Status: status,
			}, nil
		},
	)
}

// subscribeFreightApproved subscribes to FreightApproved events. Each resulting
// GenericEvent carries a Freight whose Namespace and Status.ApprovedFor identify
// the Stage the Freight was approved for, which should be reconciled.
func subscribeFreightApproved(
	ctx context.Context,
	conn *nats.Conn,
) (<-chan event.TypedGenericEvent[*kargoapi.Freight], error) {
	return subscribeFreightEvent(
		ctx,
		conn,
		kargoapi.EventTypeFreightApproved,
		func(ce cloudevents.Event) (*kargoapi.Freight, error) {
			var data kargoEvent.FreightApproved
			if err := ce.DataAs(&data); err != nil {
				return nil, err
			}
			if data.Project == "" || data.StageName == "" {
				return nil, nil
			}
			return &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: data.Project,
					Name:      data.Name,
				},
				Status: kargoapi.FreightStatus{
					ApprovedFor: map[string]kargoapi.ApprovedStage{data.StageName: {}},
				},
			}, nil
		},
	)
}

// NewFreightCreatedStageEnqueuer returns an event handler that, for each
// FreightCreated event delivered over the Freight-created channel, enqueues the
// Stages subscribed to the originating Warehouse. It handles Generic events --
// the kind source.Channel delivers -- while Create/Update/Delete are no-ops.
func NewFreightCreatedStageEnqueuer(
	kargoClient client.Client,
	forControlFlowStages bool,
) handler.TypedEventHandler[*kargoapi.Freight, reconcile.Request] {
	return handler.TypedFuncs[*kargoapi.Freight, reconcile.Request]{
		GenericFunc: func(
			ctx context.Context,
			evt event.TypedGenericEvent[*kargoapi.Freight],
			wq workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			genericFreightCreated(ctx, evt, wq, kargoClient, forControlFlowStages)
		},
	}
}

func genericFreightCreated(
	ctx context.Context,
	evt event.TypedGenericEvent[*kargoapi.Freight],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
	kargoClient client.Client,
	forControlFlowStages bool,
) {
	logger := logging.LoggerFromContext(ctx)

	freight, ok := any(evt.Object).(*kargoapi.Freight)
	if !ok {
		return
	}

	stages := kargoapi.StageList{}
	if err := kargoClient.List(
		ctx,
		&stages,
		&client.ListOptions{
			Namespace: freight.Namespace,
			FieldSelector: fields.OneTermEqualSelector(
				indexer.StagesByWarehouseField,
				freight.Origin.Name,
			),
		},
	); err != nil {
		logger.Error(
			err, "Failed to list Stages subscribed to Warehouse",
			"warehouse", freight.Origin.Name,
			"namespace", freight.Namespace,
		)
		return
	}

	for _, stage := range stages.Items {
		if stage.IsControlFlow() != forControlFlowStages {
			continue
		}
		enqueueStage(ctx, wq, freight.Namespace, stage.Name, forControlFlowStages)
	}
}

// NewFreightVerifiedStageEnqueuer returns an event handler that, for each
// FreightVerifiedInStage event, enqueues the Stages downstream of the Stage in
// which the Freight was verified. It handles Generic events -- the kind
// source.Channel delivers -- while Create/Update/Delete are no-ops.
func NewFreightVerifiedStageEnqueuer(
	kargoClient client.Client,
	forControlFlowStages bool,
) handler.TypedEventHandler[*kargoapi.Freight, reconcile.Request] {
	return handler.TypedFuncs[*kargoapi.Freight, reconcile.Request]{
		GenericFunc: func(
			ctx context.Context,
			evt event.TypedGenericEvent[*kargoapi.Freight],
			wq workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			genericFreightVerified(ctx, evt, wq, kargoClient, forControlFlowStages)
		},
	}
}

// genericFreightVerified handles a FreightVerifiedInStage event by enqueuing
// the Stages downstream of each Stage the Freight was verified in.
func genericFreightVerified(
	ctx context.Context,
	evt event.TypedGenericEvent[*kargoapi.Freight],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
	kargoClient client.Client,
	forControlFlowStages bool,
) {
	freight := evt.Object
	if freight == nil {
		return
	}
	enqueueDownstreamStages(
		ctx,
		kargoClient,
		wq,
		freight.Namespace,
		slices.Sorted(maps.Keys(freight.Status.VerifiedIn)),
		forControlFlowStages,
	)
}

// NewFreightCurrentlyInStageEnqueuer returns an event handler that, for each
// FreightCurrentlyInStage event, enqueues the Stages downstream of the Stage the
// Freight is now current in, so a downstream Stage using the MatchUpstream
// auto-promotion selection policy re-evaluates its upstream's current Freight. It
// handles Generic events -- the kind source.Channel delivers -- while
// Create/Update/Delete are no-ops.
func NewFreightCurrentlyInStageEnqueuer(
	kargoClient client.Client,
	forControlFlowStages bool,
) handler.TypedEventHandler[*kargoapi.Freight, reconcile.Request] {
	return handler.TypedFuncs[*kargoapi.Freight, reconcile.Request]{
		GenericFunc: func(
			ctx context.Context,
			evt event.TypedGenericEvent[*kargoapi.Freight],
			wq workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			genericFreightCurrentlyInStage(ctx, evt, wq, kargoClient, forControlFlowStages)
		},
	}
}

// genericFreightCurrentlyInStage handles a FreightCurrentlyInStage event. It
// filters the event on verification -- enqueuing downstream Stages only for
// Stages the Freight is both current in and verified in -- because downstream
// availability (and the MatchUpstream selection policy) requires the upstream's
// current Freight to be verified. An unverified "currently in" event enqueues
// nothing.
func genericFreightCurrentlyInStage(
	ctx context.Context,
	evt event.TypedGenericEvent[*kargoapi.Freight],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
	kargoClient client.Client,
	forControlFlowStages bool,
) {
	freight := evt.Object
	if freight == nil {
		return
	}
	var verifiedCurrentStages []string
	for stage := range freight.Status.CurrentlyIn {
		if _, verified := freight.Status.VerifiedIn[stage]; verified {
			verifiedCurrentStages = append(verifiedCurrentStages, stage)
		}
	}
	slices.Sort(verifiedCurrentStages)
	enqueueDownstreamStages(
		ctx,
		kargoClient,
		wq,
		freight.Namespace,
		verifiedCurrentStages,
		forControlFlowStages,
	)
}

// NewFreightApprovedStageEnqueuer returns an event handler that, for each
// FreightApproved event, enqueues the Stage the Freight was approved for. It
// handles Generic events -- the kind source.Channel delivers -- while
// Create/Update/Delete are no-ops.
func NewFreightApprovedStageEnqueuer() handler.TypedEventHandler[*kargoapi.Freight, reconcile.Request] {
	return handler.TypedFuncs[*kargoapi.Freight, reconcile.Request]{
		GenericFunc: genericFreightApproved,
	}
}

// genericFreightApproved handles a FreightApproved event by enqueuing each
// Stage the Freight was approved for.
func genericFreightApproved(
	ctx context.Context,
	evt event.TypedGenericEvent[*kargoapi.Freight],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	freight := evt.Object
	if freight == nil {
		return
	}
	for _, stage := range slices.Sorted(maps.Keys(freight.Status.ApprovedFor)) {
		enqueueStage(ctx, wq, freight.Namespace, stage, false)
	}
}

// enqueueDownstreamStages enqueues the Stages downstream of each of the given
// upstream Stages, filtered to control-flow or regular Stages by
// forControlFlowStages and de-duplicated.
func enqueueDownstreamStages(
	ctx context.Context,
	kargoClient client.Client,
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
	namespace string,
	upstreamStages []string,
	forControlFlowStages bool,
) {
	logger := logging.LoggerFromContext(ctx)

	downstream := map[string]struct{}{}
	for _, upstream := range upstreamStages {
		stages := kargoapi.StageList{}
		if err := kargoClient.List(
			ctx,
			&stages,
			&client.ListOptions{
				Namespace: namespace,
				FieldSelector: fields.OneTermEqualSelector(
					indexer.StagesByUpstreamStagesField,
					upstream,
				),
			},
		); err != nil {
			logger.Error(
				err, "Failed to list downstream Stages",
				"stage", upstream,
				"namespace", namespace,
			)
			return
		}
		for _, stage := range stages.Items {
			if stage.IsControlFlow() != forControlFlowStages {
				continue
			}
			downstream[stage.Name] = struct{}{}
		}
	}

	for _, name := range slices.Sorted(maps.Keys(downstream)) {
		enqueueStage(ctx, wq, namespace, name, forControlFlowStages)
	}
}

// enqueueStage adds a reconcile request for the named Stage.
func enqueueStage(
	ctx context.Context,
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
	namespace, name string,
	forControlFlowStages bool,
) {
	wq.Add(reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: namespace, Name: name},
	})
	logging.LoggerFromContext(ctx).Debug(
		"enqueued Stage for reconciliation",
		"namespace", namespace,
		"stage", name,
		"controlFlow", forControlFlowStages,
	)
}
