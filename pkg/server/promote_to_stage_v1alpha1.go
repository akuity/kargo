package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/event"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/kargo"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/user"
)

// PromoteToStage creates a Promotion resource to transition a specified Stage
// into the state represented by the specified Freight.
func (s *server) PromoteToStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.PromoteToStageRequest],
) (*connect.Response[svcv1alpha1.PromoteToStageResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	stageName := req.Msg.GetStage()
	if err := validateFieldNotEmpty("stage", stageName); err != nil {
		return nil, err
	}

	freightName := req.Msg.GetFreight()
	freightAlias := req.Msg.GetFreightAlias()
	if (freightName == "" && freightAlias == "") || (freightName != "" && freightAlias != "") {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("exactly one of freight or freightAlias should not be empty"),
		)
	}

	if err := s.validateProjectExistsFn(ctx, project); err != nil {
		return nil, err
	}

	stage, err := s.getStageFn(
		ctx,
		s.client,
		types.NamespacedName{
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
			fmt.Errorf(
				"Stage %q not found in namespace %q",
				stageName,
				project,
			),
		)
	}

	freight, err := s.getFreightByNameOrAliasFn(
		ctx,
		s.client,
		project,
		freightName,
		freightAlias,
	)
	if err != nil {
		return nil, fmt.Errorf("get freight: %w", err)
	}
	if freight == nil {
		if freightName != "" {
			err = fmt.Errorf("freight %q not found in namespace %q", freightName, project)
		} else {
			err = fmt.Errorf("freight with alias %q not found in namespace %q", freightAlias, project)
		}
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if !s.isFreightAvailableFn(stage, freight) {
		// nolint:staticcheck
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf(
				"Freight %q is not available to Stage %q",
				freightName,
				stageName,
			),
		)
	}

	if err = s.authorizeFn(
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

	promotion, err := kargo.NewPromotionBuilder(s.client).Build(ctx, *stage, freight.Name)
	if err != nil {
		return nil, fmt.Errorf("build promotion: %w", err)
	}
	if err := s.createPromotionFn(ctx, promotion); err != nil {
		return nil, fmt.Errorf("create promotion: %w", err)
	}
	s.recordPromotionCreatedEvent(ctx, promotion, freight)
	return connect.NewResponse(&svcv1alpha1.PromoteToStageResponse{
		Promotion: promotion,
	}), nil
}

func (s *server) isFreightAvailable(
	stage *kargoapi.Stage,
	freight *kargoapi.Freight,
) bool {
	return stage.IsFreightAvailable(freight)
}

func (s *server) recordPromotionCreatedEvent(
	ctx context.Context,
	p *kargoapi.Promotion,
	f *kargoapi.Freight,
) {
	var actor string
	msg := fmt.Sprintf("Promotion created for Stage %q", p.Spec.Stage)
	if u, ok := user.InfoFromContext(ctx); ok {
		actor = api.FormatEventUserActor(u)
		msg += fmt.Sprintf(" by %q", actor)
	}

	evt := event.NewPromotionCreated(msg, actor, p, f)
	if err := s.sender.Send(ctx, evt); err != nil {
		logging.LoggerFromContext(ctx).Error(err, "Error when publishing new promotion event")
	}
}

// promoteToStageRequest represents the request body for the PromoteToStage REST endpoint.
type promoteToStageRequest struct {
	Freight      string `json:"freight,omitempty"`
	FreightAlias string `json:"freightAlias,omitempty"`
} // @name PromoteToStageRequest

// @id PromoteToStage
// @Summary Promote to Stage
// @Description Create a Promotion resource to transition a specified Stage into
// @Description the state represented by the specified Freight.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Param body body promoteToStageRequest true "Promote request"
// @Success 201 {object} object "Promotion resource (github.com/akuity/kargo/api/v1alpha1.Promotion)"
// @Router /v1beta1/projects/{project}/stages/{stage}/promotions [post]
func (s *server) promoteToStage(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	stageName := c.Param("stage")

	var req promoteToStageRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	// Validate that exactly one of freight or freightAlias is provided
	if (req.Freight == "" && req.FreightAlias == "") || (req.Freight != "" && req.FreightAlias != "") {
		_ = c.Error(libhttp.ErrorStr(
			"exactly one of freight or freightAlias must be provided",
			http.StatusBadRequest,
		))
		return
	}

	// Get the Stage
	stage := &kargoapi.Stage{}
	if err := s.client.Get(ctx, client.ObjectKey{Namespace: project, Name: stageName}, stage); err != nil {
		if apierrors.IsNotFound(err) {
			_ = c.Error(libhttp.ErrorStr(
				fmt.Sprintf("Stage %q not found in project %q", stageName, project),
				http.StatusNotFound,
			))
			return
		}
		_ = c.Error(err)
		return
	}

	// Get the Freight by name or alias
	var freight *kargoapi.Freight
	if req.Freight != "" {
		freight = &kargoapi.Freight{}
		if err := s.client.Get(ctx, client.ObjectKey{Namespace: project, Name: req.Freight}, freight); err != nil {
			if apierrors.IsNotFound(err) {
				_ = c.Error(libhttp.ErrorStr(
					fmt.Sprintf("Freight %q not found in project %q", req.Freight, project),
					http.StatusNotFound,
				))
				return
			}
			_ = c.Error(err)
			return
		}
	} else {
		// Search by alias
		list := &kargoapi.FreightList{}
		if err := s.client.List(
			ctx,
			list,
			client.InNamespace(project),
			client.MatchingLabels{kargoapi.LabelKeyAlias: req.FreightAlias},
		); err != nil {
			_ = c.Error(err)
			return
		}
		if len(list.Items) == 0 {
			_ = c.Error(libhttp.ErrorStr(
				fmt.Sprintf("Freight with alias %q not found in project %q", req.FreightAlias, project),
				http.StatusNotFound,
			))
			return
		}
		freight = &list.Items[0]
	}

	// Validate that the Freight is available to the Stage
	if !stage.IsFreightAvailable(freight) {
		_ = c.Error(libhttp.ErrorStr(
			fmt.Sprintf("Freight %q is not available to Stage %q", freight.Name, stageName),
			http.StatusBadRequest,
		))
		return
	}

	// Build and create the Promotion
	promotion, err := kargo.NewPromotionBuilder(s.client).Build(ctx, *stage, freight.Name)
	if err != nil {
		_ = c.Error(fmt.Errorf("build promotion: %w", err))
		return
	}

	if err := s.client.Create(ctx, promotion); err != nil {
		_ = c.Error(fmt.Errorf("create promotion: %w", err))
		return
	}

	if s.sender != nil {
		s.recordPromotionCreatedEvent(ctx, promotion, freight)
	}

	c.JSON(http.StatusCreated, gin.H{"promotion": promotion})
}
