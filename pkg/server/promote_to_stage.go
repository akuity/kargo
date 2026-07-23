package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/event"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/user"
)

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
	// Origin is the canonical Freight origin key (e.g. "Warehouse/foo"). When
	// set, the promotion webhook resolves it to the current auto-promotion
	// candidate. Exactly one of Freight, FreightAlias, or Origin must be set.
	Origin string `json:"origin,omitempty"`
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
// @Success 201 {object} kargoapi.Promotion "Promotion resource (github.com/akuity/kargo/api/v1alpha1.Promotion)"
// @Router /v1beta1/projects/{project}/stages/{stage}/promotions [post]
func (s *server) promoteToStage(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	stageName := c.Param("stage")

	var req promoteToStageRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	nonEmpty := 0
	for _, v := range []string{req.Freight, req.FreightAlias, req.Origin} {
		if v != "" {
			nonEmpty++
		}
	}
	if nonEmpty != 1 {
		_ = c.Error(libhttp.ErrorStr(
			"exactly one of freight, freightAlias, or origin must be provided",
			http.StatusBadRequest,
		))
		return
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

	if err = s.authorizeFn(
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

	if req.Origin != "" {
		// Let admission resolve the origin to the auto-promotion candidate
		// Freight. That keeps "promote by origin" race-free for REST clients.
		origin, parseErr := kargoapi.ParseFreightOrigin(req.Origin)
		if parseErr != nil {
			_ = c.Error(libhttp.ErrorStr(
				fmt.Sprintf("invalid origin %q: %s", req.Origin, parseErr),
				http.StatusBadRequest,
			))
			return
		}
		promotion := api.NewMinimalPromotionForOrigin(stage, origin)
		if u, ok := user.InfoFromContext(ctx); ok {
			api.SetCreateActorAnnotation(promotion, api.FormatEventUserActor(u))
		}
		if err = s.createPromotionFn(ctx, promotion); err != nil {
			_ = c.Error(err)
			return
		}
		if s.sender != nil && promotion.Spec.Freight != "" {
			freight := &kargoapi.Freight{}
			if err = s.client.Get(
				ctx,
				client.ObjectKey{Namespace: promotion.Namespace, Name: promotion.Spec.Freight},
				freight,
			); err != nil {
				logging.LoggerFromContext(ctx).Error(
					err,
					"error getting resolved Freight for Promotion created event",
				)
			} else {
				s.recordPromotionCreatedEvent(ctx, promotion, freight)
			}
		}
		c.JSON(http.StatusCreated, promotion)
		return
	}

	var freight *kargoapi.Freight
	if req.Freight != "" {
		freight = &kargoapi.Freight{}
		if err = s.client.Get(ctx, client.ObjectKey{Namespace: project, Name: req.Freight}, freight); err != nil {
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
		list := &kargoapi.FreightList{}
		if err = s.client.List(
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

	if !stage.IsFreightAvailable(freight) {
		_ = c.Error(libhttp.ErrorStr(
			fmt.Sprintf("Freight %q is not available to Stage %q", freight.Name, stageName),
			http.StatusBadRequest,
		))
		return
	}

	// Create the Promotion. The defaulting webhook fills in the rest from
	// the Stage's PromotionTemplate.
	promotion := api.NewMinimalPromotion(stage, freight.Name)
	if u, ok := user.InfoFromContext(ctx); ok {
		api.SetCreateActorAnnotation(promotion, api.FormatEventUserActor(u))
	}

	if err := s.createPromotionFn(ctx, promotion); err != nil {
		_ = c.Error(err)
		return
	}

	if s.sender != nil {
		s.recordPromotionCreatedEvent(ctx, promotion, freight)
	}

	c.JSON(http.StatusCreated, promotion)
}
