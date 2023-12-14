package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) UpdateWarehouse(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateWarehouseRequest],
) (*connect.Response[svcv1alpha1.UpdateWarehouseResponse], error) {
	var warehouse kargoapi.Warehouse
	switch {
	case req.Msg.GetYaml() != "":
		if err := yaml.Unmarshal([]byte(req.Msg.GetYaml()), &warehouse); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "invalid yaml"))
		}
	case req.Msg.GetTyped() != nil:
		warehouse = kargoapi.Warehouse{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Msg.GetTyped().GetProject(),
				Name:      req.Msg.GetTyped().GetName(),
			},
			Spec: typesv1alpha1.FromWarehouseSpecProto(req.Msg.GetTyped().GetSpec()),
		}
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("warehouse should not be empty"))
	}

	if err := validateProjectAndWarehouseName(warehouse.GetNamespace(), warehouse.GetName()); err != nil {
		return nil, err
	}
	if err := s.validateProject(ctx, warehouse.GetNamespace()); err != nil {
		return nil, err
	}
	var existingWarehouse kargoapi.Warehouse
	if err := s.client.Get(ctx, client.ObjectKeyFromObject(&warehouse), &existingWarehouse); err != nil {
		if kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeUnknown, errors.Wrap(err, "get warehouse"))
	}
	warehouse.SetResourceVersion(existingWarehouse.GetResourceVersion())
	if err := s.client.Update(ctx, &warehouse); err != nil {
		return nil, connect.NewError(connect.CodeUnknown, errors.Wrap(err, "update warehouse"))
	}
	return connect.NewResponse(&svcv1alpha1.UpdateWarehouseResponse{
		Warehouse: typesv1alpha1.ToWarehouseProto(warehouse),
	}), nil
}
