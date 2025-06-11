package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (s *server) WatchClusterConfig(
	ctx context.Context,
	_ *connect.Request[svcv1alpha1.WatchClusterConfigRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchClusterConfigResponse],
) error {
	const name = "cluster" // TODO(hidde): Define this in the (internal) API?

	if err := s.client.Get(ctx, client.ObjectKey{
		Name: name,
	}, &kargoapi.ClusterConfig{}); err != nil {
		return fmt.Errorf("get ClusterConfig: %w", err)
	}

	opts := metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, name).String(),
	}
	w, err := s.client.Watch(ctx, &kargoapi.ClusterConfig{}, name, opts)
	if err != nil {
		return fmt.Errorf("watch ClusterConfig: %w", err)
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
			var config *kargoapi.ClusterConfig
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &config); err != nil {
				return fmt.Errorf("from unstructured: %w", err)
			}
			if err := stream.Send(&svcv1alpha1.WatchClusterConfigResponse{
				ClusterConfig: config,
				Type:          string(e.Type),
			}); err != nil {
				return fmt.Errorf("send response: %w", err)
			}
		}
	}
}
