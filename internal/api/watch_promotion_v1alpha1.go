package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) WatchPromotion(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.WatchPromotionRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchPromotionResponse],
) error {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return err
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return err
	}

	if err := s.client.Get(ctx, client.ObjectKey{
		Namespace: project,
		Name:      name,
	}, &kargoapi.Promotion{}); err != nil {
		return fmt.Errorf("get promotion: %w", err)
	}

	opts := metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, name).String(),
	}
	w, err := s.client.Watch(ctx, &kargoapi.Promotion{}, project, opts)
	if err != nil {
		return fmt.Errorf("watch promotion: %w", err)
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
				return fmt.Errorf("unexpected object type %T", e.Object)
			}
			var promotion *kargoapi.Promotion
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &promotion); err != nil {
				return fmt.Errorf("from unstructured: %w", err)
			}
			if err := stream.Send(&svcv1alpha1.WatchPromotionResponse{
				Promotion: promotion,
				Type:      string(e.Type),
			}); err != nil {
				return fmt.Errorf("send response: %w", err)
			}
		}
	}
}
