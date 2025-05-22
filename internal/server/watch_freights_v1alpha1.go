package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func (s *server) WatchFreights(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.WatchFreightsRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchFreightsResponse],
) error {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return err
	}

	opts := metav1.ListOptions{} // No field selectors for now, watching all Freights
	w, err := s.client.Watch(ctx, &kargoapi.Freight{}, project, opts)
	if err != nil {
		return fmt.Errorf("watch freight: %w", err)
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
			var freight *kargoapi.Freight
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &freight); err != nil {
				return fmt.Errorf("from unstructured: %w", err)
			}
			if err := stream.Send(&svcv1alpha1.WatchFreightsResponse{
				Freight: freight,
				Type:    string(e.Type),
			}); err != nil {
				return fmt.Errorf("send response: %w", err)
			}
		}
	}
}
