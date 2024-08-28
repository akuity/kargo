package api

import (
	"context"
	"fmt"
	"slices"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) ListProjectEvents(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListProjectEventsRequest],
) (*connect.Response[svcv1alpha1.ListProjectEventsResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	var eventsList corev1.EventList
	if err := s.client.List(
		ctx,
		&eventsList,
		client.InNamespace(req.Msg.GetProject()),
		// List Kargo related events only
		client.MatchingFieldsSelector{
			Selector: fields.OneTermEqualSelector(
				kubeclient.EventsByInvolvedObjectAPIGroupIndexField,
				kargoapi.GroupVersion.Group,
			),
		},
	); err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	// Sort descending by last timestamp
	slices.SortFunc(eventsList.Items, func(lhs, rhs corev1.Event) int {
		return rhs.LastTimestamp.Time.Compare(lhs.LastTimestamp.Time)
	})

	events := make([]*corev1.Event, len(eventsList.Items))
	for i := range eventsList.Items {
		events[i] = &eventsList.Items[i]
	}

	return connect.NewResponse(&svcv1alpha1.ListProjectEventsResponse{
		Events: events,
	}), nil
}
