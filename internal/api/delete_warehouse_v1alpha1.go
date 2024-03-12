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

func (s *server) DeleteWarehouse(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteWarehouseRequest],
) (*connect.Response[svcv1alpha1.DeleteWarehouseResponse], error) {
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

	var warehouse kargoapi.Warehouse
	key := client.ObjectKey{
		Namespace: project,
		Name:      name,
	}
	if err := s.client.Get(ctx, key, &warehouse); err != nil {
		if kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("warehouse %q not found", key.String()))
		}
		return nil, fmt.Errorf("get warehouse: %w", err)
	}
	if err := s.client.Delete(ctx, &warehouse); err != nil && !kubeerr.IsNotFound(err) {
		return nil, fmt.Errorf("delete warehouse: %w", err)
	}
	return connect.NewResponse(&svcv1alpha1.DeleteWarehouseResponse{}), nil
}
