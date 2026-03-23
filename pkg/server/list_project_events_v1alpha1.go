package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
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
				indexer.EventsByInvolvedObjectAPIGroupField,
				kargoapi.GroupVersion.Group,
			),
		},
	); err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	// Sort descending by last timestamp
	slices.SortFunc(eventsList.Items, func(lhs, rhs corev1.Event) int {
		return rhs.LastTimestamp.Compare(lhs.LastTimestamp.Time)
	})

	events := make([]*corev1.Event, len(eventsList.Items))
	for i := range eventsList.Items {
		events[i] = &eventsList.Items[i]
	}

	return connect.NewResponse(&svcv1alpha1.ListProjectEventsResponse{
		Events: events,
	}), nil
}

// @id ListProjectEvents
// @Summary List project-level Kubernetes Events
// @Description List Kubernetes Events from a project's namespace. Returns a
// @Description Kubernetes EventList resource.
// @Tags Events, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Produce json
// @Success 200 {object} object "EventList resource (k8s.io/api/core/v1.EventList)"
// @Router /v1beta1/projects/{project}/events [get]
func (s *server) listProjectEvents(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")

	list := &corev1.EventList{}
	if err := s.client.List(
		ctx,
		list,
		client.InNamespace(project),
		// List Kargo related events only
		client.MatchingFieldsSelector{
			Selector: fields.OneTermEqualSelector(
				indexer.EventsByInvolvedObjectAPIGroupField,
				kargoapi.GroupVersion.Group,
			),
		},
	); err != nil {
		_ = c.Error(err)
		return
	}

	// Sort descending by last timestamp
	slices.SortFunc(list.Items, func(lhs, rhs corev1.Event) int {
		return rhs.LastTimestamp.Compare(lhs.LastTimestamp.Time)
	})

	c.JSON(http.StatusOK, list)
}
