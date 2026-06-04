package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	if err = rejectedFreightError(freight, "promoted"); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
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

	result, err := s.createStagePromotion(
		ctx,
		stage,
		freight,
		stagePromotionOptions{},
	)
	if err != nil {
		return nil, stagePromotionConnectError(err)
	}
	promotion := result.Promotion
	s.recordPromotionCreatedEvent(ctx, promotion, freight)
	if result.CreatedHold {
		if _, err = api.RefreshStage(
			ctx,
			s.client.InternalClient(),
			client.ObjectKey{Namespace: project, Name: stageName},
		); err != nil {
			logging.LoggerFromContext(ctx).Error(
				err,
				"error refreshing Stage after creating rollback Promotion",
				"stage", stageName,
				"promotion", promotion.Name,
			)
		}
	}
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
	if s.sender == nil {
		return
	}

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
	Freight               string `json:"freight,omitempty"`
	FreightAlias          string `json:"freightAlias,omitempty"`
	ExpectedAutoCandidate string `json:"expectedAutoCandidate,omitempty"`
	Reason                string `json:"reason,omitempty"`
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
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Failure 504 {object} ErrorResponse
// @Router /v1beta1/projects/{project}/stages/{stage}/promotions [post]
func (s *server) promoteToStage(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	stageName := c.Param("stage")
	key := client.ObjectKey{Namespace: project, Name: stageName}

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
	req.Reason = strings.TrimSpace(req.Reason)
	if len(req.Reason) > 1024 {
		_ = c.Error(libhttp.ErrorStr(
			"reason cannot be longer than 1024 characters",
			http.StatusBadRequest,
		))
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

	// Get the Stage
	stage, ok := s.getRESTStage(ctx, project, stageName, c)
	if !ok {
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

	if err := rejectedFreightError(freight, "promoted"); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusConflict))
		return
	}

	// Validate that the Freight is available to the Stage
	if !stage.IsFreightAvailable(freight) {
		_ = c.Error(libhttp.ErrorStr(
			fmt.Sprintf("Freight %q is not available to Stage %q", freight.Name, stageName),
			http.StatusBadRequest,
		))
		return
	}

	result, err := s.createStagePromotion(
		ctx,
		stage,
		freight,
		stagePromotionOptions{
			ExpectedAutoCandidate: req.ExpectedAutoCandidate,
			Reason:                req.Reason,
		},
	)
	if err != nil {
		_ = c.Error(stagePromotionRESTError(err))
		return
	}
	promotion := result.Promotion

	if s.sender != nil {
		s.recordPromotionCreatedEvent(ctx, promotion, freight)
	}
	if result.CreatedHold {
		// The caller was authorized for the custom "promote" verb above.
		// The internal client performs the mechanical refresh annotation
		// write because users are not generally allowed to patch Stages.
		if _, err = api.RefreshStage(ctx, s.client.InternalClient(), key); err != nil {
			logging.LoggerFromContext(ctx).Error(
				err,
				"error refreshing Stage after creating rollback Promotion",
				"stage", stageName,
				"promotion", promotion.Name,
			)
		}
	}

	c.JSON(http.StatusCreated, promotion)
}

// stagePromotionOptions carries optional REST-only safeguards for creating a
// Promotion directly to a Stage.
type stagePromotionOptions struct {
	ExpectedAutoCandidate string
	Reason                string
}

// stagePromotionResult names the Promotion and side effect produced by
// createStagePromotion.
type stagePromotionResult struct {
	Promotion   *kargoapi.Promotion
	CreatedHold bool
}

// stagePromotionConflictError reports user-retryable conflicts detected before
// creating a Promotion.
type stagePromotionConflictError struct {
	message string
}

func (s *stagePromotionConflictError) Error() string {
	return s.message
}

func newStagePromotionConflictError(format string, args ...any) error {
	return &stagePromotionConflictError{message: fmt.Sprintf(format, args...)}
}

// createStagePromotion creates a non-auto Promotion and, when the selected
// Freight is older than the current auto-promotion candidate, first records a
// pending auto-promotion hold. If Promotion creation fails after the hold write,
// deterministic API errors remove the hold immediately; ambiguous create errors
// leave it in place for Stage controller recovery.
func (s *server) createStagePromotion(
	ctx context.Context,
	stage *kargoapi.Stage,
	freight *kargoapi.Freight,
	opts stagePromotionOptions,
) (stagePromotionResult, error) {
	promotion, err := kargo.NewPromotionBuilder(s.client).Build(ctx, *stage, freight.Name)
	if err != nil {
		return stagePromotionResult{}, fmt.Errorf("build promotion: %w", err)
	}

	key := client.ObjectKey{Namespace: stage.Namespace, Name: stage.Name}
	if err = s.authorizeFn(
		ctx,
		"create",
		kargoapi.GroupVersion.WithResource("promotions"),
		"",
		client.ObjectKeyFromObject(promotion),
	); err != nil {
		return stagePromotionResult{}, err
	}

	candidate, err := s.getAutoPromotionCandidate(ctx, stage, freight.Origin)
	if err != nil {
		return stagePromotionResult{}, fmt.Errorf("get auto-promotion candidate: %w", err)
	}
	if err = expectedAutoCandidateConflict(opts.ExpectedAutoCandidate, candidate); err != nil {
		return stagePromotionResult{}, err
	}

	createdHold := false
	var createdPendingHold kargoapi.AutoPromotionHold
	if candidate != nil && candidate.Name != freight.Name {
		var existingHold *kargoapi.AutoPromotionHold
		var candidateErr error
		var candidateConflict error
		now := metav1.Now()
		hold := kargoapi.AutoPromotionHold{
			Freight: kargoapi.FreightReference{
				Name:   freight.Name,
				Origin: freight.Origin,
			},
			State:         kargoapi.AutoPromotionHoldStatePending,
			PromotionName: promotion.Name,
			Actor:         autoPromotionHoldActor(ctx),
			Reason:        opts.Reason,
			CreatedAt:     &now,
		}
		// The status patch callback can only report whether it changed the
		// Stage. Capture conflict details so callers still receive precise
		// 409s instead of a generic "unchanged" result.
		if createdHold, err = s.patchStageAutoPromotionHoldsWithStage(ctx, key, func(liveStage *kargoapi.Stage) bool {
			existingHold = nil
			candidateErr = nil
			candidateConflict = nil

			liveCandidate, liveCandidateErr := s.getAutoPromotionCandidate(ctx, liveStage, freight.Origin)
			if liveCandidateErr != nil {
				candidateErr = fmt.Errorf("get live auto-promotion candidate: %w", liveCandidateErr)
				return false
			}
			if candidateConflict = expectedAutoCandidateConflict(
				opts.ExpectedAutoCandidate,
				liveCandidate,
			); candidateConflict != nil {
				return false
			}
			if liveCandidate == nil || liveCandidate.Name == freight.Name {
				return false
			}

			if statusHold, ok := liveStage.Status.GetAutoPromotionHold(freight.Origin); ok {
				existing := statusHold
				existingHold = &existing
				return false
			}
			return upsertAutoPromotionHold(&liveStage.Status, freight.Origin, hold)
		}); err != nil {
			return stagePromotionResult{}, fmt.Errorf("create auto-promotion hold: %w", err)
		}
		if candidateErr != nil {
			return stagePromotionResult{}, candidateErr
		}
		if candidateConflict != nil {
			return stagePromotionResult{}, candidateConflict
		}
		if existingHold != nil {
			return stagePromotionResult{}, newStagePromotionConflictError(
				"auto-promotion is already %s for origin %q; wait for the "+
					"current rollback to settle or resume auto-promotion before "+
					"creating another rollback",
				strings.ToLower(string(existingHold.State)),
				freight.Origin.String(),
			)
		}
		if createdHold {
			createdPendingHold = hold
			annotateRollbackPromotion(promotion)
		}
	} else if candidate != nil && candidate.Name == freight.Name {
		hold, held := stage.Status.GetAutoPromotionHold(freight.Origin)
		liveStage := &kargoapi.Stage{}
		if err = s.client.InternalClient().Get(ctx, key, liveStage); err != nil {
			return stagePromotionResult{}, fmt.Errorf("get live Stage before clearing auto-promotion hold: %w", err)
		}
		liveCandidate, err := s.getAutoPromotionCandidate(ctx, liveStage, freight.Origin)
		if err != nil {
			return stagePromotionResult{}, fmt.Errorf("get live auto-promotion candidate: %w", err)
		}
		if err = expectedAutoCandidateConflict(opts.ExpectedAutoCandidate, liveCandidate); err != nil {
			return stagePromotionResult{}, err
		}
		if liveCandidate != nil && liveCandidate.Name != freight.Name {
			return stagePromotionResult{}, newStagePromotionConflictError(
				"auto-promotion candidate changed to %q; reload and try again",
				liveCandidate.Name,
			)
		}

		liveHold, liveHeld := liveStage.Status.GetAutoPromotionHold(freight.Origin)
		if liveHeld && liveHold.State == kargoapi.AutoPromotionHoldStatePending {
			return stagePromotionResult{}, newStagePromotionConflictError(
				"auto-promotion is pending for origin %q; wait for the "+
					"current rollback to settle before promoting the current candidate",
				freight.Origin.String(),
			)
		}
		if liveHeld {
			if !held ||
				liveHold.State != kargoapi.AutoPromotionHoldStateActive ||
				hold.State != kargoapi.AutoPromotionHoldStateActive ||
				!api.AutoPromotionHoldIdentityMatches(liveHold, hold) {
				return stagePromotionResult{}, newStagePromotionConflictError(
					"auto-promotion hold for origin %q changed; reload and try again",
					freight.Origin.String(),
				)
			}
			annotateClearAutoPromotionHold(promotion, freight.Origin, liveHold)
		} else if held {
			return stagePromotionResult{}, newStagePromotionConflictError(
				"auto-promotion hold for origin %q changed; reload and try again",
				freight.Origin.String(),
			)
		}
	}

	createPromotionFn := s.createPromotionFn
	if createPromotionFn == nil {
		createPromotionFn = s.client.Create
	}
	if createErr := createPromotionFn(ctx, promotion); createErr != nil {
		if createdHold && !promotionCreateMayHavePersisted(createErr) {
			if _, cleanupErr := s.patchStageAutoPromotionHolds(ctx, key, func(status *kargoapi.StageStatus) bool {
				return removeAutoPromotionHolds(status, func(origin string, hold kargoapi.AutoPromotionHold) bool {
					return origin == freight.Origin.String() &&
						autoPromotionHoldMatchesPendingCreate(hold, createdPendingHold)
				})
			}); cleanupErr != nil {
				return stagePromotionResult{}, apierrors.NewInternalError(
					fmt.Errorf("create promotion: %w; cleanup pending auto-promotion hold: %w", createErr, cleanupErr),
				)
			}
		}
		return stagePromotionResult{}, createPromotionError(createErr)
	}
	return stagePromotionResult{
		Promotion:   promotion,
		CreatedHold: createdHold,
	}, nil
}

// autoPromotionHoldMatchesPendingCreate checks that a pending hold is exactly
// the hold this request created using fields that cannot change through API
// serialization. The Promotion name is unique for this request.
func autoPromotionHoldMatchesPendingCreate(
	hold kargoapi.AutoPromotionHold,
	expected kargoapi.AutoPromotionHold,
) bool {
	return hold.State == kargoapi.AutoPromotionHoldStatePending &&
		hold.PromotionName == expected.PromotionName &&
		hold.Freight.Name == expected.Freight.Name &&
		hold.Freight.Origin.Equals(&expected.Freight.Origin)
}

func annotateRollbackPromotion(promotion *kargoapi.Promotion) {
	if promotion.Annotations == nil {
		promotion.Annotations = make(map[string]string, 1)
	}
	promotion.Annotations[kargoapi.AnnotationKeyRollback] = kargoapi.AnnotationValueTrue
}

// expectedAutoCandidateConflict returns a conflict when a request's
// stale-candidate precondition no longer matches the current candidate.
func expectedAutoCandidateConflict(
	expected string,
	candidate *kargoapi.Freight,
) error {
	if expected == "" {
		return nil
	}
	currentCandidate := ""
	if candidate != nil {
		currentCandidate = candidate.Name
	}
	if currentCandidate == expected {
		return nil
	}
	return newStagePromotionConflictError(
		"auto-promotion candidate changed from %q to %q; reload and try again",
		expected,
		currentCandidate,
	)
}

func stagePromotionRESTError(err error) error {
	var conflictErr *stagePromotionConflictError
	if errors.As(err, &conflictErr) {
		return libhttp.ErrorStr(conflictErr.Error(), http.StatusConflict)
	}
	return err
}

func stagePromotionConnectError(err error) error {
	var conflictErr *stagePromotionConflictError
	if errors.As(err, &conflictErr) {
		return connect.NewError(connect.CodeFailedPrecondition, conflictErr)
	}
	return err
}

func annotateClearAutoPromotionHold(
	promotion *kargoapi.Promotion,
	origin kargoapi.FreightOrigin,
	hold kargoapi.AutoPromotionHold,
) {
	if promotion.Annotations == nil {
		promotion.Annotations = make(map[string]string, 4)
	}
	promotion.Annotations[kargoapi.AnnotationKeyClearAutoPromotionHold] = origin.String()
	promotion.Annotations[kargoapi.AnnotationKeyClearAutoPromotionHoldPromotion] = hold.PromotionName
	promotion.Annotations[kargoapi.AnnotationKeyClearAutoPromotionHoldPromotionUID] = hold.PromotionUID
	if hold.CreatedAt != nil {
		promotion.Annotations[kargoapi.AnnotationKeyClearAutoPromotionHoldCreatedAt] =
			hold.CreatedAt.Format(time.RFC3339Nano)
	}
}

// promotionCreateMayHavePersisted reports whether a create error may have been
// returned after the API server persisted the Promotion. Ambiguous errors keep
// the pending hold so controller recovery can reconcile the partial state.
func promotionCreateMayHavePersisted(err error) bool {
	if apierrors.IsAlreadyExists(err) {
		return true
	}
	if apierrors.IsTimeout(err) ||
		apierrors.IsServerTimeout(err) ||
		apierrors.IsServiceUnavailable(err) ||
		apierrors.IsInternalError(err) ||
		apierrors.IsUnexpectedServerError(err) {
		return true
	}
	var statusErr *apierrors.StatusError
	return !errors.As(err, &statusErr)
}

func createPromotionError(err error) error {
	var statusErr *apierrors.StatusError
	if errors.As(err, &statusErr) {
		status := statusErr.ErrStatus
		status.Message = fmt.Sprintf("create promotion: %s", status.Message)
		return &apierrors.StatusError{ErrStatus: status}
	}
	return apierrors.NewInternalError(fmt.Errorf("create promotion: %w", err))
}
