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

func (s *server) WatchStages(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.WatchStagesRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchStagesResponse],
) error {
	if req.Msg.GetProject() == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
	}
	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return err
	}

	if req.Msg.GetName() != "" {
		if err := s.client.Get(ctx, libClient.ObjectKey{
			Namespace: req.Msg.GetProject(),
			Name:      req.Msg.GetName(),
		}, &kargoapi.Stage{}); err != nil {
			if kubeerr.IsNotFound(err) {
				return connect.NewError(connect.CodeNotFound, err)
			}
			return errors.Wrap(err, "get stage")
		}
	}

	opts := metav1.ListOptions{}
	if req.Msg.GetName() != "" {
		opts.FieldSelector = fields.OneTermEqualSelector(metav1.ObjectNameField, req.Msg.GetName()).String()
	}
	w, err :=
		s.client.Watch(ctx, &kargoapi.Stage{}, req.Msg.GetProject(), opts)
	if err != nil {
		return errors.Wrap(err, "watch stage")
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
			var stage *kargoapi.Stage
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &stage); err != nil {
				return errors.Wrap(err, "from unstructured")
			}
			if err := stream.Send(&svcv1alpha1.WatchStagesResponse{
				Stage: stage,
				Type:  string(e.Type),
			}); err != nil {
				return errors.Wrap(err, "send response")
			}
		}
	}
}
