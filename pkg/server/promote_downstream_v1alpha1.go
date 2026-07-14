package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/user"
)

// findDownstreamStages returns a list of Stages that are immediately downstream
// from the given Stage and request Freight from the given origin.
// TODO: this could be powered by an index.
func (s *server) findDownstreamStages(
	ctx context.Context,
	stage *kargoapi.Stage,
	origin kargoapi.FreightOrigin,
) ([]kargoapi.Stage, error) {
	var allStages kargoapi.StageList
	if err := s.client.List(ctx, &allStages, client.InNamespace(stage.Namespace)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var downstreams []kargoapi.Stage
	for _, s := range allStages.Items {
		for _, req := range s.Spec.RequestedFreight {
			if !req.Origin.Equals(&origin) {
				continue
			}
			for _, upstream := range req.Sources.Stages {
				if upstream == stage.Name {
					downstreams = append(downstreams, s)
				}
			}
		}
	}
	return downstreams, nil
}

// promoteDownstreamRequest represents the request body for the PromoteDownstream REST endpoint.
type promoteDownstreamRequest struct {
	Freight      string `json:"freight,omitempty"`
	FreightAlias string `json:"freightAlias,omitempty"`
} // @name PromoteDownstreamRequest

// @id PromoteDownstream
// @Summary Promote downstream
// @Description Creates a Promotion resource for each of a Stage's immediately
// @Description downstream Stages.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Param body body promoteDownstreamRequest true "Promote request"
// @Success 201 {object} object "Promotions created"
// @Router /v1beta1/projects/{project}/stages/{stage}/promotions/downstream [post]
func (s *server) promoteDownstream(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	stageName := c.Param("stage")

	var req promoteDownstreamRequest
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

	// Find downstream stages
	downstreams, err := s.findDownstreamStages(ctx, stage, freight.Origin)
	if err != nil {
		_ = c.Error(fmt.Errorf("find downstream stages: %w", err))
		return
	}

	if len(downstreams) == 0 {
		_ = c.Error(libhttp.ErrorStr(
			fmt.Sprintf("Stage %q has no downstream stages", stageName),
			http.StatusNotFound,
		))
		return
	}

	for _, downstream := range downstreams {
		if err := s.authorizeFn(
			ctx,
			"promote",
			kargoapi.GroupVersion.WithResource("stages"),
			"",
			types.NamespacedName{
				Namespace: downstream.Namespace,
				Name:      downstream.Name,
			},
		); err != nil {
			_ = c.Error(err)
			return
		}
	}

	// Validate that freight is available to all downstream stages
	for _, downstream := range downstreams {
		if !downstream.IsFreightAvailable(freight) {
			_ = c.Error(libhttp.ErrorStr(
				fmt.Sprintf("Freight %q is not available to downstream Stage %q", freight.Name, downstream.Name),
				http.StatusBadRequest,
			))
			return
		}
	}

	// Create promotions for all downstream stages
	var actor string
	if u, ok := user.InfoFromContext(ctx); ok {
		actor = api.FormatEventUserActor(u)
	}

	promoteErrs := make([]error, 0, len(downstreams))
	createdPromos := make([]*kargoapi.Promotion, 0, len(downstreams))

	for _, downstream := range downstreams {
		// Skip "control flow" stages with no promotion steps
		if downstream.Spec.PromotionTemplate != nil &&
			len(downstream.Spec.PromotionTemplate.Spec.Steps) == 0 {
			continue
		}

		newPromo := api.NewMinimalPromotion(&downstream, freight.Name)
		if actor != "" {
			api.SetCreateActorAnnotation(newPromo, actor)
		}

		if err := s.client.Create(ctx, newPromo); err != nil {
			promoteErrs = append(promoteErrs, err)
			continue
		}

		if s.sender != nil {
			s.recordPromotionCreatedEvent(ctx, newPromo, freight)
		}
		createdPromos = append(createdPromos, newPromo)
	}

	response := gin.H{"promotions": createdPromos}
	if len(promoteErrs) > 0 {
		response["errors"] = errors.Join(promoteErrs...).Error()
		c.JSON(http.StatusMultiStatus, response)
		return
	}

	c.JSON(http.StatusCreated, response)
}
