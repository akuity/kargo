package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) GetWarehouse(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetWarehouseRequest],
) (*connect.Response[svcv1alpha1.GetWarehouseResponse], error) {
	if err := validateProjectAndWarehouseName(req.Msg.GetProject(), req.Msg.GetName()); err != nil {
		return nil, err
	}
	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return nil, err
	}

	var warehouse kargoapi.Warehouse
	if err := s.client.Get(ctx, client.ObjectKey{
		Namespace: req.Msg.GetProject(),
		Name:      req.Msg.GetName(),
	}, &warehouse); err != nil {
		if kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, errors.Wrap(err, "get warehouse")
	}
	return connect.NewResponse(&svcv1alpha1.GetWarehouseResponse{
		Warehouse: typesv1alpha1.ToWarehouseProto(warehouse),
	}), nil
}
