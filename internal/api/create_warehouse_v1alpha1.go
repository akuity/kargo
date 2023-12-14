package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) CreateWarehouse(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateWarehouseRequest],
) (*connect.Response[svcv1alpha1.CreateWarehouseResponse], error) {
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
	if err := s.client.Create(ctx, &warehouse); err != nil {
		if kubeerr.IsAlreadyExists(err) {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		return nil, connect.NewError(connect.CodeUnknown, errors.Wrap(err, "create warehouse"))
	}
	return connect.NewResponse(&svcv1alpha1.CreateWarehouseResponse{
		Warehouse: typesv1alpha1.ToWarehouseProto(warehouse),
	}), nil
}
