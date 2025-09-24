package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (s *server) WatchPromotions(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.WatchPromotionsRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchPromotionsResponse],
) error {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return err
	}

	stage := req.Msg.GetStage()

	if stage != "" {
		if err := s.client.Get(ctx, client.ObjectKey{
			Namespace: project,
			Name:      stage,
		}, &kargoapi.Stage{}); err != nil {
			return fmt.Errorf("get stage: %w", err)
		}
	}

	w, err := s.client.Watch(ctx, &kargoapi.Promotion{}, project, metav1.ListOptions{})
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
			if err = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &promotion); err != nil {
				return fmt.Errorf("from unstructured: %w", err)
			}
			// FIXME: Current (dynamic) client doesn't support filtering with indexed field by indexer,
			// so manually filter stage here.
			if stage != "" && stage != promotion.Spec.Stage {
				continue
			}
			if err = stream.Send(&svcv1alpha1.WatchPromotionsResponse{
				Promotion: promotion,
				Type:      string(e.Type),
			}); err != nil {
				return fmt.Errorf("send response: %w", err)
			}
		}
	}
}
