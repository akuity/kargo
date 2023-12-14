package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) DeleteWarehouse(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteWarehouseRequest],
) (*connect.Response[svcv1alpha1.DeleteWarehouseResponse], error) {
	if err := validateProjectAndWarehouseName(req.Msg.GetProject(), req.Msg.GetName()); err != nil {
		return nil, err
	}
	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return nil, err
	}
	var warehouse kargoapi.Warehouse
	key := client.ObjectKey{
		Namespace: req.Msg.GetProject(),
		Name:      req.Msg.GetName(),
	}
	if err := s.client.Get(ctx, key, &warehouse); err != nil {
		if kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("warehouse %q not found", key.String()))
		}
		return nil, connect.NewError(connect.CodeUnknown, errors.Wrap(err, "get warehouse"))
	}
	if err := s.client.Delete(ctx, &warehouse); err != nil && !kubeerr.IsNotFound(err) {
		return nil, connect.NewError(connect.CodeUnknown, errors.Wrap(err, "delete warehouse"))
	}
	return connect.NewResponse(&svcv1alpha1.DeleteWarehouseResponse{}), nil
}
