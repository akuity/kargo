package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/event"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/kubeclient"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/user"
)

func (s *server) ApproveFreight(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ApproveFreightRequest],
) (*connect.Response[svcv1alpha1.ApproveFreightResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	alias := req.Msg.GetAlias()
	if (name == "" && alias == "") ||
		(name != "" && alias != "") {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("exactly one of name or alias should not be empty"),
		)
	}

	stageName := req.Msg.GetStage()
	if err := validateFieldNotEmpty("stage", stageName); err != nil {
		return nil, err
	}

	if err := s.validateProjectExistsFn(ctx, project); err != nil {
		return nil, err
	}

	freight, err := s.getFreightByNameOrAliasFn(
		ctx,
		s.client,
		project,
		name,
		alias,
	)
	if err != nil {
		return nil, fmt.Errorf("get freight: %w", err)
	}
	if freight == nil {
		if name != "" {
			err = fmt.Errorf("freight %q not found in namespace %q", name, project)
		} else {
			err = fmt.Errorf("freight with alias %q not found in namespace %q", alias, project)
		}
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	stage, err := s.getStageFn(
		ctx,
		s.client,
		client.ObjectKey{
			Namespace: project,
			Name:      stageName,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get stage: %w", err)
	}
	if stage == nil {
		// nolint:staticcheck
		return nil, connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf("Stage %q not found in namespace %q", stageName, project),
		)
	}

	if err := s.authorizeFn(
		ctx,
		"promote",
		schema.GroupVersionResource{
			Group:    kargoapi.GroupVersion.Group,
			Version:  kargoapi.GroupVersion.Version,
			Resource: "stages",
		},
		"",
		types.NamespacedName{
			Namespace: project,
			Name:      stageName,
		},
	); err != nil {
		return nil, err
	}

	if freight.IsApprovedFor(stageName) {
		return &connect.Response[svcv1alpha1.ApproveFreightResponse]{}, nil
	}

	newStatus := *freight.Status.DeepCopy()
	if newStatus.ApprovedFor == nil {
		newStatus.ApprovedFor = make(map[string]kargoapi.ApprovedStage)
	}
	newStatus.AddApprovedStage(stageName, time.Now())

	if err := s.patchFreightStatusFn(ctx, freight, newStatus); err != nil {
		return nil, fmt.Errorf("patch status: %w", err)
	}

	var actor string
	eventMsg := fmt.Sprintf("Freight approved for Stage %q", stageName)
	if u, ok := user.InfoFromContext(ctx); ok {
		actor = api.FormatEventUserActor(u)
		eventMsg += fmt.Sprintf(" by %q", actor)
	}

	evt := event.NewFreightApproved(eventMsg, actor, stageName, freight)
	if err := s.sender.Send(ctx, evt); err != nil {
		logging.LoggerFromContext(ctx).Error(err,
			"error sending Freight approved event")
	}
	return &connect.Response[svcv1alpha1.ApproveFreightResponse]{}, nil
}

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
