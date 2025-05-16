package server

import (
	"context"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/internal/api"
)

func (s *server) RefreshWarehouse(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.RefreshWarehouseRequest],
) (*connect.Response[svcv1alpha1.RefreshWarehouseResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	warehouse, err := api.RefreshWarehouse(ctx, s.client.InternalClient(), client.ObjectKey{
		Namespace: project,
		Name:      name,
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&svcv1alpha1.RefreshWarehouseResponse{
		Warehouse: warehouse,
	}), nil
}
