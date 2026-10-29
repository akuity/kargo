package promotionrollouts

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/user"
)

// @id CreatePromotionRollout
// @Summary Create a PromotionRollout
// @Description Create a PromotionRollout, which promotes the Freight named
// @Description in the body to each of the Targets the Stage governs, or to
// @Description the ones the body names among them. PromotionRollouts are
// @Description managed through this API rather than as Kubernetes
// @Description resources. The Stage must select Targets: its selectors are
// @Description resolved against the project's Targets as the rollout is
// @Description created, narrowed to the Targets named if any are, and its
// @Description update strategy is resolved against how many that leaves;
// @Description both are recorded with the rollout for good. A Target the
// @Description Stage does not govern is refused. Everything else about the
// @Description rollout is assigned by the server.
// @Description This is a promotion: it requires the promote verb on the
// @Description Stage. A server that does not create PromotionRollouts
// @Description responds 501.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Param body body createPromotionRolloutRequest true "Create request"
// @Success 201 {object} kargoapi.PromotionRollout "PromotionRollout resource"
// @Router /v1beta1/projects/{project}/stages/{stage}/promotion-rollouts [post]
func (h *Handler) create(c *gin.Context) {
	req, ok := bindCreateRequest(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	project := c.Param(paramProject)
	stageName := c.Param(paramStage)

	stage, err := api.GetStage(
		ctx,
		h.kube,
		types.NamespacedName{Namespace: project, Name: stageName},
	)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if stage == nil {
		_ = c.Error(libhttp.ErrorStr(
			fmt.Sprintf("Stage %q not found in project %q", stageName, project),
			http.StatusNotFound,
		))
		return
	}

	// Creating a rollout is promoting to its Stage, so the promote verb on
	// the Stage is the authorization decision. It is checked once the Stage
	// is known to exist, as the promote endpoints check it.
	if err = h.authorize(ctx, can.Promote().Stage(project, stageName)); err != nil {
		_ = c.Error(err)
		return
	}

	if !stage.IsTargetAware() {
		_ = c.Error(libhttp.ErrorStr(
			fmt.Sprintf(
				"Stage %q does not select Targets; promote to it with a Promotion instead",
				stageName,
			),
			http.StatusBadRequest,
		))
		return
	}

	freight, err := api.GetFreight(
		ctx,
		h.kube,
		types.NamespacedName{Namespace: project, Name: req.Freight},
	)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if freight == nil {
		_ = c.Error(libhttp.ErrorStr(
			fmt.Sprintf("Freight %q not found in project %q", req.Freight, project),
			http.StatusNotFound,
		))
		return
	}
	if !stage.IsFreightAvailable(freight) {
		_ = c.Error(libhttp.ErrorStr(
			fmt.Sprintf("Freight %q is not available to Stage %q", freight.Name, stageName),
			http.StatusBadRequest,
		))
		return
	}

	var actor string
	if u, ok := user.IdentityFromContext(ctx); ok {
		actor = u.Actor()
	}
	var rollout *kargoapi.PromotionRollout
	if rollout, err = h.creator.Create(ctx, stage, freight, req.Targets, actor); err != nil {
		_ = c.Error(err)
		return
	}
	logChange(ctx, "PromotionRollout created", project, rollout.Name)
	c.JSON(http.StatusCreated, rollout)
}
