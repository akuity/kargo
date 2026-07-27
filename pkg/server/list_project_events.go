package server

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
)

// @id ListProjectEvents
// @Summary List project-level Kubernetes Events
// @Description List Kubernetes Events from a project's namespace. Returns a
// @Description Kubernetes EventList resource.
// @Tags Events, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Produce json
// @Success 200 {object} corev1.EventList "EventList resource (k8s.io/api/core/v1.EventList)"
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
