package server

import (
	"context"
	"errors"
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
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/request"
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
	if u, ok := user.IdentityFromContext(ctx); ok {
		actor = u.Actor()
		msg += fmt.Sprintf(" by %q", actor)
	}

	evt, err := event.NewPromotionCreated(msg, actor, p, f)
	if err == nil {
		err = s.sender.Send(ctx, evt)
	}
	if err != nil {
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
	// Targets is a list of target names to promote to. If set to `nil` - selects all targets for the stage.
	// If non-nil, should be a subset of targets selected by the stage, empty list is an error.
	Targets []string `json:"targets,omitempty"`
	// UpdateStrategy configures overrides for maxConcurrent and maxErrors for this request.
	// If nil - calculated from the stages updateStrategy based on the number of stage targets.
	UpdateStrategy *kargoapi.PromotionRequestUpdateStrategy `json:"updateStrategy,omitempty"`
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
// @Description A Stage that selects Targets yields a PromotionRequest, which fans
// @Description the Freight out to each of them, in place of a single Promotion.
// @Success 201 {object} kargoapi.Promotion "Promotion resource, or a PromotionRequest for a Stage that selects Targets"
// @Router /v1beta1/projects/{project}/stages/{stage}/promotions [post]
func (s *server) promoteToStage(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	stageName := c.Param("stage")

	var req promoteToStageRequest
	if !request.BindJSON(c, &req) {
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

	if err = s.authorize(ctx, can.Promote().Stage(project, stageName)); err != nil {
		_ = c.Error(err)
		return
	}

	if req.Origin != "" {
		if stage.IsTargetAware() {
			// Promotion by origin relies on the Promotion defaulting webhook to
			// resolve the origin to a candidate Freight at admission time.
			// PromotionRequests have no such webhook, so there is nothing to resolve
			// the origin. Refuse rather than silently create a Promotion that
			// bypasses the Stage's Targets.
			_ = c.Error(libhttp.ErrorStr(
				fmt.Sprintf(
					"promotion by origin is not supported for Stage %q, which selects Targets",
					stageName,
				),
				http.StatusBadRequest,
			))
			return
		}
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
		if u, ok := user.IdentityFromContext(ctx); ok {
			api.SetCreateActorAnnotation(promotion, u.Actor())
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
		if freight, err = s.getFreightByAlias(ctx, project, req.FreightAlias); err != nil {
			_ = c.Error(err)
			return
		}
	}

	if !stage.IsFreightAvailable(freight) {
		_ = c.Error(libhttp.ErrorStr(
			fmt.Sprintf("Freight %q is not available to Stage %q", freight.Name, stageName),
			http.StatusBadRequest,
		))
		return
	}

	// A Stage that selects Targets fans Freight out to them via a PromotionRequest
	// rather than promoting to itself with a single Promotion.
	if stage.IsTargetAware() {
		// We explicitly request all targets if req.Targets is nil
		// TODO: split up API functions to avoid confusion between nil argument and empty argument
		allTargets := req.Targets == nil
		if !allTargets && len(req.Targets) == 0 {
			_ = c.Error(libhttp.Error(fmt.Errorf("target list cannot be empty"), http.StatusBadRequest))
			return
		}
		newPromoReq, promoErr := s.createPromotionRequest(
			ctx, stage, freight, allTargets, req.Targets, req.UpdateStrategy)
		if promoErr != nil {
			if errors.Is(promoErr, api.ErrTargetNotSelected) {
				_ = c.Error(libhttp.Error(promoErr, http.StatusBadRequest))
			} else {
				_ = c.Error(promoErr)
			}
			return
		}
		c.JSON(http.StatusCreated, newPromoReq)
		return
	}

	// Create the Promotion. The defaulting webhook fills in the rest from
	// the Stage's PromotionTemplate.
	newPromo, err := s.createPromotion(ctx, stage, freight)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, newPromo)
}

func (s *server) createPromotion(
	ctx context.Context,
	stage *kargoapi.Stage,
	freight *kargoapi.Freight,
) (*kargoapi.Promotion, error) {
	newPromo := api.NewMinimalPromotion(stage, freight.Name)

	var actor string
	if u, ok := user.IdentityFromContext(ctx); ok {
		actor = u.Actor()
	}
	if actor != "" {
		api.SetCreateActorAnnotation(newPromo, actor)
	}
	// FIXME: resolve name conflict
	if err := s.createPromotionFn(ctx, newPromo); err != nil {
		return nil, err
	}
	if s.sender != nil {
		s.recordPromotionCreatedEvent(ctx, newPromo, freight)
	}
	return newPromo, nil
}

func (s *server) createPromotionRequest(
	ctx context.Context,
	stage *kargoapi.Stage,
	freight *kargoapi.Freight,
	allTargets bool,
	targets []string,
	updateStrategy *kargoapi.PromotionRequestUpdateStrategy,
) (*kargoapi.PromotionRequest, error) {
	// The create goes through the internal client: PromotionRequests are
	// system-owned, and the promote-verb check above IS the authorization
	// decision for this request. The Targets come from the database
	// without a check of their own for the same reason.

	listTargetsFn, err := s.getListTargetsFn()
	if err != nil {
		return nil, err
	}

	var newPromoReq *kargoapi.PromotionRequest
	if allTargets {
		newPromoReq, err = api.NewPromotionRequestForAllTargets(
			ctx, stage, freight.Name, listTargetsFn, updateStrategy)
	} else {
		// Build promotionRequestTargets from targets arg.
		// If allTargets is false deliberately builds an empty list.
		promotionRequestTargets := make([]kargoapi.PromotionRequestTarget, len(targets))
		for i, targetName := range targets {
			promotionRequestTargets[i] = kargoapi.PromotionRequestTarget{Name: targetName}
		}
		newPromoReq, err = api.NewPromotionRequestForSelectTargets(
			ctx, stage, freight.Name, listTargetsFn, updateStrategy, promotionRequestTargets)
	}

	if err != nil {
		return nil, err
	}

	if u, ok := user.IdentityFromContext(ctx); ok {
		api.SetCreateActorAnnotation(newPromoReq, u.Actor())
	}
	if err = s.client.InternalClient().Create(ctx, newPromoReq); err != nil {
		return nil, err
	}

	// No event is recorded: Kargo's promotion events carry a Promotion, and
	// a PromotionRequest has none of its own.
	// TODO: 1125 record events here

	return newPromoReq, nil
}
