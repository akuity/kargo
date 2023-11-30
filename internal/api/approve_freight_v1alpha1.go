package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) ApproveFreight(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ApproveFreightRequest],
) (*connect.Response[svcv1alpha1.ApproveFreightResponse], error) {
	if err := validateProjectAndStageNonEmpty(req.Msg.GetProject(), req.Msg.GetStage()); err != nil {
		return nil, err
	}
	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return nil, err
	}

	var stage kargoapi.Stage
	key := client.ObjectKey{
		Namespace: req.Msg.GetProject(),
		Name:      req.Msg.GetStage(),
	}
	if err := s.client.Get(ctx, key, &stage); err != nil {
		if kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("stage %q not found", key.String()))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var freight kargoapi.Freight
	freightKey := client.ObjectKey{
		Namespace: req.Msg.GetProject(),
		Name:      req.Msg.GetId(),
	}
	if err := s.client.Get(ctx, freightKey, &freight); err != nil {
		if kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("freight %q not found", freightKey.String()))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var st any = stage
	approvedStage, ok := st.(kargoapi.ApprovedStage)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("could not cast stage %q to approved stage", key.String()))
	}
	freight.Status.ApprovedFor[req.Msg.GetStage()] = approvedStage

	if err := s.client.Update(ctx, &freight); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return &connect.Response[svcv1alpha1.ApproveFreightResponse]{}, nil
}
