package api

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/user"
	"github.com/akuity/kargo/internal/event"
	"github.com/akuity/kargo/internal/kargo"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
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
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf(
				"Freight %q is not available to Stage %q",
				freightName,
				stageName,
			),
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

	promotion := kargo.NewPromotion(ctx, *stage, freight.Name)
	if err := s.createPromotionFn(ctx, &promotion); err != nil {
		return nil, fmt.Errorf("create promotion: %w", err)
	}
	s.recordPromotionCreatedEvent(ctx, &promotion, freight)
	return connect.NewResponse(&svcv1alpha1.PromoteToStageResponse{
		Promotion: &promotion,
	}), nil
}

func (s *server) recordPromotionCreatedEvent(
	ctx context.Context,
	p *kargoapi.Promotion,
	f *kargoapi.Freight,
) {
	var actor string
	msg := fmt.Sprintf("Promotion created for Stage %q", p.Spec.Stage)
	if u, ok := user.InfoFromContext(ctx); ok {
		actor = event.FormatEventUserActor(u)
		msg += fmt.Sprintf(" by %q", actor)
	}

	s.recorder.AnnotatedEventf(
		p,
		event.NewPromotionEventAnnotations(ctx, actor, p, f),
		corev1.EventTypeNormal,
		event.EventReasonPromotionCreated,
		msg,
	)
}
