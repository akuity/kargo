package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

func (s *server) WatchProjectConfig(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.WatchProjectConfigRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchProjectConfigResponse],
) error {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return err
	}

	if err := s.client.Get(ctx, client.ObjectKey{
		Namespace: project,
		Name:      project,
	}, &kargoapi.ProjectConfig{}); err != nil {
		return fmt.Errorf("get projectconfig: %w", err)
	}

	w, err := s.client.Watch(
		ctx,
		&kargoapi.ProjectConfigList{},
		client.InNamespace(project),
		client.MatchingFields{"metadata.name": project},
	)
	if err != nil {
		return fmt.Errorf("watch ProjectConfig: %w", err)
	}
	defer w.Stop()
	for {
		select {
		case <-ctx.Done():
			logger := logging.LoggerFromContext(ctx)
			logger.Debug(ctx.Err().Error())
			return nil
		case e, ok := <-w.ResultChan():
			if !ok {
				return nil
			}
			u, ok := e.Object.(*unstructured.Unstructured)
			if !ok {
				return fmt.Errorf("unexpected object type %T", e.Object)
			}
			var config *kargoapi.ProjectConfig
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &config); err != nil {
				return fmt.Errorf("from unstructured: %w", err)
			}
			if err := stream.Send(&svcv1alpha1.WatchProjectConfigResponse{
				ProjectConfig: config,
				Type:          string(e.Type),
			}); err != nil {
				return fmt.Errorf("send response: %w", err)
			}
		}
	}
}
