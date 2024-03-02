package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) ApproveFreight(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ApproveFreightRequest],
) (*connect.Response[svcv1alpha1.ApproveFreightResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	freightID := req.Msg.GetId()
	if err := validateFieldNotEmpty("id", freightID); err != nil {
		return nil, err
	}

	stageName := req.Msg.GetStage()
	if err := validateFieldNotEmpty("stage", stageName); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	var freight kargoapi.Freight
	freightKey := client.ObjectKey{
		Namespace: project,
		Name:      freightID,
	}
	if err := s.client.Get(ctx, freightKey, &freight); err != nil {
		if kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("freight %q not found", freightKey.String()))
		}
		return nil, errors.Wrap(err, "get freight")
	}

	var stage kargoapi.Stage
	key := client.ObjectKey{
		Namespace: project,
		Name:      stageName,
	}
	if err := s.client.Get(ctx, key, &stage); err != nil {
		if kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("stage %q not found", key.String()))
		}
		return nil, errors.Wrap(err, "get stage")
	}

	newStatus := *freight.Status.DeepCopy()
	if newStatus.ApprovedFor == nil {
		newStatus.ApprovedFor = map[string]kargoapi.ApprovedStage{}
	}

	if _, ok := newStatus.ApprovedFor[stageName]; ok {
		return &connect.Response[svcv1alpha1.ApproveFreightResponse]{}, nil
	}

	newStatus.ApprovedFor[stageName] = kargoapi.ApprovedStage{}

	if err := kubeclient.PatchStatus(
		ctx,
		s.client,
		&freight,
		func(status *kargoapi.FreightStatus) {
			*status = newStatus
		},
	); err != nil {
		return nil, errors.Wrap(err, "patch status")
	}

	return &connect.Response[svcv1alpha1.ApproveFreightResponse]{}, nil
}
