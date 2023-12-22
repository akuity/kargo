package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) WatchPromotions(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.WatchPromotionsRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchPromotionsResponse],
) error {
	if req.Msg.GetProject() == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
	}
	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return err
	}

	if req.Msg.GetStage() != "" {
		if err := s.client.Get(ctx, client.ObjectKey{
			Namespace: req.Msg.GetProject(),
			Name:      req.Msg.GetStage(),
		}, &kargoapi.Stage{}); err != nil {
			if kubeerr.IsNotFound(err) {
				return connect.NewError(connect.CodeNotFound, err)
			}
			return errors.Wrap(err, "get stage")
		}
	}

	w, err := s.client.Watch(ctx, &kargoapi.Promotion{}, req.Msg.GetProject(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "watch promotion")
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
				return connect.NewError(connect.CodeInternal, errors.Errorf("unexpected object type %T", e.Object))
			}
			var promotion *kargoapi.Promotion
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &promotion); err != nil {
				return connect.NewError(connect.CodeInternal, errors.Wrap(err, "from unstructured"))
			}
			// FIXME: Current (dynamic) client doesn't support filtering with indexed field by indexer,
			// so manually filter stage here.
			if req.Msg.GetStage() != "" && req.Msg.GetStage() != promotion.Spec.Stage {
				continue
			}
			if err := stream.Send(&svcv1alpha1.WatchPromotionsResponse{
				Promotion: typesv1alpha1.ToPromotionProto(*promotion),
				Type:      string(e.Type),
			}); err != nil {
				return connect.NewError(connect.CodeInternal, errors.Wrap(err, "send response"))
			}
		}
	}
}
