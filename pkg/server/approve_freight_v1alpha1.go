package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/event"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/kubeclient"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/user"
)

func (s *server) patchFreightStatus(
	ctx context.Context,
	freight *kargoapi.Freight,
	newStatus kargoapi.FreightStatus,
) error {
	if err := kubeclient.PatchStatus(
		ctx,
		s.client,
		freight,
		func(status *kargoapi.FreightStatus) {
			*status = newStatus
		},
	); err != nil {
		return fmt.Errorf(
			"error patching Freight %q status in namespace %q: %w",
			freight.Name,
			freight.Namespace,
			err,
		)
	}
	return nil
}

// @id ApproveFreight
// @Summary Approve Freight for promotion to a Stage
// @Description Approve Freight for promotion to a Stage.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param freight-name-or-alias path string true "Freight name or alias"
// @Param stage query string true "Stage name"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/freight/{freight-name-or-alias}/approve [post]
func (s *server) approveFreight(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	freightNameOrAlias := c.Param("freight-name-or-alias")
	stageName := c.Query("stage")

	if stageName == "" {
		_ = c.Error(libhttp.Error(
			errors.New("stage query parameter is required"),
			http.StatusBadRequest,
		))
		return
	}

	freight := s.getFreightByNameOrAliasForGin(c, project, freightNameOrAlias)
	if freight == nil {
		return
	}

	stage := &kargoapi.Stage{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Name: stageName, Namespace: project},
		stage,
	); err != nil {
		_ = c.Error(err)
		return
	}

	if err := s.authorizeFn(
		ctx,
		"promote",
		kargoapi.GroupVersion.WithResource("stages"),
		"",
		types.NamespacedName{
			Namespace: project,
			Name:      stageName,
		},
	); err != nil {
		_ = c.Error(err)
		return
	}

	// Approval bypasses verification and soak requirements, but not origin
	// membership: an approval for a Stage that doesn't request Freight from the
	// Freight's origin could never be acted upon and is certainly a mistake.
	if !stage.RequestsFreightFromOrigin(freight.Origin) {
		_ = c.Error(libhttp.ErrorStr(
			fmt.Sprintf(
				"Stage %q does not request Freight from origin %q",
				stageName, freight.Origin.String(),
			),
			http.StatusBadRequest,
		))
		return
	}

	if freight.IsApprovedFor(stageName) {
		c.Status(http.StatusOK)
		return
	}

	newStatus := *freight.Status.DeepCopy()
	if newStatus.ApprovedFor == nil {
		newStatus.ApprovedFor = make(map[string]kargoapi.ApprovedStage)
	}
	newStatus.AddApprovedStage(stageName, time.Now())

	if err := kubeclient.PatchStatus(
		ctx,
		s.client,
		freight,
		func(status *kargoapi.FreightStatus) {
			*status = newStatus
		},
	); err != nil {
		_ = c.Error(fmt.Errorf("patch freight status: %w", err))
		return
	}

	var actor string
	eventMsg := fmt.Sprintf("Freight approved for Stage %q", stageName)
	if u, ok := user.InfoFromContext(ctx); ok {
		actor = api.FormatEventUserActor(u)
		eventMsg += fmt.Sprintf(" by %q", actor)
	}

	if s.sender != nil {
		evt := event.NewFreightApproved(eventMsg, actor, stageName, freight)
		if err := s.sender.Send(ctx, evt); err != nil {
			logging.LoggerFromContext(ctx).Error(err,
				"error sending Freight approved event")
		}
	}

	c.Status(http.StatusOK)
}
