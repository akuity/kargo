package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) ListWarehouses(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListWarehousesRequest],
) (*connect.Response[svcv1alpha1.ListWarehousesResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	var list kargoapi.WarehouseList
	if err := s.client.List(ctx, &list, client.InNamespace(project)); err != nil {
		return nil, fmt.Errorf("list warehouses: %w", err)
	}

	warehouses := make([]*kargoapi.Warehouse, len(list.Items))
	for idx := range list.Items {
		warehouses[idx] = &list.Items[idx]
	}
	return connect.NewResponse(&svcv1alpha1.ListWarehousesResponse{
		Warehouses: warehouses,
	}), nil
}
