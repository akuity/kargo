package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	libClient "sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/logging"
)

func (s *server) WatchStages(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.WatchStagesRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchStagesResponse],
) error {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return err
	}

	name := req.Msg.GetName()
	warehouses := req.Msg.GetFreightOrigins()
	summary := req.Msg.GetSummary()

	if name != "" {
		if err := s.client.Get(ctx, libClient.ObjectKey{
			Namespace: project,
			Name:      name,
		}, &kargoapi.Stage{}); err != nil {
			return fmt.Errorf("get stage: %w", err)
		}
	}

	watchOpts := []libClient.ListOption{libClient.InNamespace(project)}
	if name != "" {
		watchOpts = append(watchOpts, libClient.MatchingFields{"metadata.name": name})
	}
	if rv := req.Msg.GetResourceVersion(); rv != "" {
		watchOpts = append(watchOpts, &libClient.ListOptions{
			Raw: &metav1.ListOptions{ResourceVersion: rv},
		})
	}
	w, err := s.client.Watch(ctx, &kargoapi.StageList{}, watchOpts...)
	if err != nil {
		return fmt.Errorf("watch stage: %w", err)
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
			stage, ok := e.Object.(*kargoapi.Stage)
			if !ok {
				return fmt.Errorf("unexpected object type %T", e.Object)
			}
			if len(warehouses) > 0 && !api.StageMatchesAnyWarehouse(stage, warehouses) {
				continue
			}
			if summary {
				// StripStageForSummary mutates in place; copy first so
				// shaping the response does not depend on the watch
				// object's ownership semantics.
				stage = stage.DeepCopy()
				api.StripStageForSummary(stage)
			}
			if err := stream.Send(&svcv1alpha1.WatchStagesResponse{
				Stage: stage,
				Type:  string(e.Type),
			}); err != nil {
				return fmt.Errorf("send response: %w", err)
			}
		}
	}
}
