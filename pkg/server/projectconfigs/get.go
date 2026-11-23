package projectconfigs

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/projectconfigstore"
	natsrec "github.com/akuity/kargo/pkg/reconciler/nats"
	"github.com/akuity/kargo/pkg/server/natswatch"
)

// @id GetProjectConfig
// @Summary Retrieve ProjectConfig
// @Description Retrieve a project's ProjectConfig.
// @Tags Core, Project-Level, Config, Singleton
// @Security BearerAuth
// @Param project path string true "Project name"
// @Produce json
// @Success 200 {object} Resource "ProjectConfig resource"
// @Router /v1beta1/projects/{project}/config [get]
func (h *Handler) get(c *gin.Context) {
	project, ok := projectOf(c)
	if !ok {
		return
	}
	if c.Query("watch") == "true" {
		h.watch(c, project)
		return
	}
	cfg, err := h.load(c.Request.Context(), project)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, ResponseBody(cfg))
}

// watch streams the Project's ProjectConfig: as it is now, then again after
// every change the store announces for the Project, until it is deleted or
// the client goes away.
func (h *Handler) watch(c *gin.Context, project *kargoapi.Project) {
	natswatch.Serve(
		c,
		h.conn,
		projectconfigstore.Events,
		func(e natsrec.Event[projectconfigstore.Key, database.ProjectConfigRow]) bool {
			return e.Key.Project == project.Name
		},
		func(ctx context.Context) (*Resource, error) {
			cfg, err := h.load(ctx, project)
			if err != nil {
				return nil, err
			}
			return ResponseBody(cfg), nil
		},
		func(r *Resource) string { return r.ResourceVersion },
		watchKeepalive,
	)
}

// load reads the Project's ProjectConfig as it is served. Like the store, it
// does not find a ProjectConfig left over from an earlier Project of the same
// name. Its error is the one a Kubernetes-backed handler would have reported,
// still wrapping the store's, so that a watch can tell the ProjectConfig is
// gone.
func (h *Handler) load(ctx context.Context, project *kargoapi.Project) (*kargoapi.ProjectConfig, error) {
	cfg, err := h.store.Get(ctx, project.Name, string(project.UID))
	if err != nil {
		return nil, &translatedError{
			translated: storeError(err, project.Name),
			cause:      err,
		}
	}
	return projectconfigstore.ToAPI(cfg), nil
}
