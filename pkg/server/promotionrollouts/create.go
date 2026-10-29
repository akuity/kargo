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
// @Description Create a PromotionRollout in a project, which promotes the
// @Description Freight named by spec.freight to each of the Targets the
// @Description Stage named by spec.stage governs. PromotionRollouts are
// @Description managed through this API rather than as Kubernetes
// @Description resources. The Stage must select Targets: its selectors are
// @Description resolved against the project's Targets as the rollout is
// @Description created, its update strategy against how many they match,
// @Description and both are recorded with the rollout for good. The name,
// @Description number, Targets, update strategy and status are assigned by
// @Description the server and must not be given. This is a promotion: it
// @Description requires the promote verb on the Stage. A server that does
// @Description not create PromotionRollouts responds 501.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param body body kargoapi.PromotionRollout true "PromotionRollout resource"
// @Success 201 {object} kargoapi.PromotionRollout "PromotionRollout resource"
// @Router /v1beta1/projects/{project}/promotion-rollouts [post]
func (h *Handler) create(c *gin.Context) {
	requested, ok := bindPromotionRollout(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	project := c.Param(paramProject)

	stage, err := api.GetStage(
		ctx,
		h.kube,
		types.NamespacedName{Namespace: project, Name: requested.Spec.Stage},
	)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if stage == nil {
		_ = c.Error(libhttp.ErrorStr(
			fmt.Sprintf("Stage %q not found in project %q", requested.Spec.Stage, project),
			http.StatusNotFound,
		))
		return
	}

	// Creating a rollout is promoting to its Stage, so the promote verb on
	// the Stage is the authorization decision. The Stage comes from the
	// body, which is why this is not route middleware like the reads'.
	if err = h.authorize(ctx, can.Promote().Stage(project, requested.Spec.Stage)); err != nil {
		_ = c.Error(err)
		return
	}

	if !stage.IsTargetAware() {
		_ = c.Error(libhttp.ErrorStr(
			fmt.Sprintf(
				"Stage %q does not select Targets; promote to it with a Promotion instead",
				requested.Spec.Stage,
			),
			http.StatusBadRequest,
		))
		return
	}

	freight, err := api.GetFreight(
		ctx,
		h.kube,
		types.NamespacedName{Namespace: project, Name: requested.Spec.Freight},
	)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if freight == nil {
		_ = c.Error(libhttp.ErrorStr(
			fmt.Sprintf("Freight %q not found in project %q", requested.Spec.Freight, project),
			http.StatusNotFound,
		))
		return
	}
	if !stage.IsFreightAvailable(freight) {
		_ = c.Error(libhttp.ErrorStr(
			fmt.Sprintf("Freight %q is not available to Stage %q", freight.Name, requested.Spec.Stage),
			http.StatusBadRequest,
		))
		return
	}

	var actor string
	if u, ok := user.IdentityFromContext(ctx); ok {
		actor = u.Actor()
	}
	var rollout *kargoapi.PromotionRollout
	if rollout, err = h.creator.Create(ctx, stage, freight, actor); err != nil {
		_ = c.Error(err)
		return
	}
	logChange(ctx, "PromotionRollout created", project, rollout.Name)
	c.JSON(http.StatusCreated, rollout)
}
