package api

import (
	"context"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) RefreshWarehouse(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.RefreshWarehouseRequest],
) (*connect.Response[svcv1alpha1.RefreshWarehouseResponse], error) {
	if err := validateProjectAndWarehouseName(req.Msg.GetProject(), req.Msg.GetName()); err != nil {
		return nil, err
	}
	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return nil, err
	}

	warehouse, err := kargoapi.RefreshWarehouse(ctx, s.client, client.ObjectKey{
		Namespace: req.Msg.GetProject(),
		Name:      req.Msg.GetName(),
	})
	if err != nil {
		return nil, connect.NewError(getCodeFromError(err), err)
	}
	return connect.NewResponse(&svcv1alpha1.RefreshWarehouseResponse{
		Warehouse: typesv1alpha1.ToWarehouseProto(*warehouse),
	}), nil
}
