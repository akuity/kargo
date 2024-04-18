package api

import (
	"context"
	"fmt"
	"sort"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
	); err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	sort.Slice(eventsList.Items, func(i, j int) bool {
		return eventsList.Items[i].LastTimestamp.Time.After(eventsList.Items[j].LastTimestamp.Time)
	})

	events := make([]*corev1.Event, len(eventsList.Items))
	for i, event := range eventsList.Items {
		events[i] = &event
	}

	return connect.NewResponse(&svcv1alpha1.ListProjectEventsResponse{
		Events: events,
	}), nil
}
