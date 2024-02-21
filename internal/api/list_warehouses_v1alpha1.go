package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/api/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) ListWarehouses(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListWarehousesRequest],
) (*connect.Response[svcv1alpha1.ListWarehousesResponse], error) {
	if req.Msg.GetProject() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
	}
	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return nil, err
	}

	var list kargoapi.WarehouseList
	if err := s.client.List(ctx, &list, client.InNamespace(req.Msg.GetProject())); err != nil {
		return nil, errors.Wrap(err, "list warehouses")
	}

	warehouses := make([]*v1alpha1.Warehouse, len(list.Items))
	for idx := range list.Items {
		warehouses[idx] = &list.Items[idx]
	}
	return connect.NewResponse(&svcv1alpha1.ListWarehousesResponse{
		Warehouses: warehouses,
	}), nil
}
