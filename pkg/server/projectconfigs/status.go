package projectconfigs

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database/projectconfigstore"
	"github.com/akuity/kargo/pkg/server/projectconfigs/wire"
	"github.com/akuity/kargo/pkg/server/request"
)

// @id UpdateProjectConfigStatus
// @Summary Update ProjectConfig status
// @Description Record the outcome of a reconcile of a project's
// @Description ProjectConfig. The status is refused if the ProjectConfig,
// @Description its generation or its refresh token is no longer the one the
// @Description reconcile observed. Webhook receivers whose paths another
// @Description ProjectConfig owns are not recorded, and are named in the
// @Description response. Only the management controller is meant to call
// @Description this; it requires update on projectconfigs/status.
// @Tags Core, Project-Level, Config, Singleton
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param body body wire.StatusRequest true "Observed status"
// @Success 200 {object} wire.StatusResponse "Status recorded"
// @Router /v1beta1/projects/{project}/config/status [put]
func (h *Handler) status(c *gin.Context) {
	project, ok := projectOf(c)
	if !ok {
		return
	}
	var req wire.StatusRequest
	if !request.BindJSON(c, &req) {
		return
	}
	observedID, err := uuid.Parse(req.ObservedUID)
	if err != nil {
		_ = c.Error(invalid(project.Name, field.ErrorList{field.Invalid(
			field.NewPath("observedUID"),
			req.ObservedUID,
			"must be the metadata.uid of the observed ProjectConfig",
		)}))
		return
	}
	ctx := c.Request.Context()
	// The store writes status by Project name alone, so confirm first that
	// the Project's own ProjectConfig is the one observed. A leftover of a
	// deleted Project of the same name is not found, and must not take the
	// status or re-claim its webhook receiver paths.
	current, err := h.store.Get(ctx, project.Name, string(project.UID))
	if err != nil {
		_ = c.Error(storeError(err, project.Name))
		return
	}
	if current.ID != observedID {
		_ = c.Error(apierrors.NewConflict(
			kargoapi.GroupVersion.WithResource("projectconfigs").GroupResource(),
			project.Name,
			errors.New("the ProjectConfig was deleted and recreated"),
		))
		return
	}
	skipped, err := h.store.UpdateStatus(ctx, project.Name, projectconfigstore.Status{
		ObservedID:           observedID,
		ObservedGeneration:   req.ObservedGeneration,
		ObservedRefreshToken: req.ObservedRefreshToken,
		Conditions:           req.Conditions,
		WebhookReceivers:     req.WebhookReceivers,
	})
	if err != nil {
		_ = c.Error(storeError(err, project.Name))
		return
	}
	logChange(ctx, "ProjectConfig status updated", project.Name)
	c.JSON(http.StatusOK, wire.StatusResponse{SkippedReceivers: skipped})
}
