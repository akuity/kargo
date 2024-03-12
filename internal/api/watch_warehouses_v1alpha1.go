package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	libClient "sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) WatchWarehouses(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.WatchWarehousesRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchWarehousesResponse],
) error {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return err
	}

	name := req.Msg.GetName()

	if name != "" {
		if err := s.client.Get(ctx, libClient.ObjectKey{
			Namespace: project,
			Name:      name,
		}, &kargoapi.Warehouse{}); err != nil {
			if kubeerr.IsNotFound(err) {
				return connect.NewError(connect.CodeNotFound, err)
			}
			return errors.Wrap(err, "get warehouse")
		}
	}

	opts := metav1.ListOptions{}
	if name != "" {
		opts.FieldSelector = fields.OneTermEqualSelector(metav1.ObjectNameField, name).String()
	}
	w, err :=
		s.client.Watch(ctx, &kargoapi.Warehouse{}, project, opts)
	if err != nil {
		return errors.Wrap(err, "watch warehouse")
	}
	defer w.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e, ok := <-w.ResultChan():
			if !ok {
				return nil
			}
			u, ok := e.Object.(*unstructured.Unstructured)
			if !ok {
				return errors.Errorf("unexpected object type %T", e.Object)
			}
			var warehouse *kargoapi.Warehouse
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &warehouse); err != nil {
				return errors.Wrap(err, "from unstructured")
			}
			if err := stream.Send(&svcv1alpha1.WatchWarehousesResponse{
				Warehouse: warehouse,
				Type:      string(e.Type),
			}); err != nil {
				return errors.Wrap(err, "send response")
			}
		}
	}
}
