package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	libClient "sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

// WatchStageSummaries streams StageSummary events for changes to Stages in
// the given project. The response payload mirrors the WatchStages endpoint
// but returns the lightweight StageSummary projection instead of the full
// Stage resource. See ListStageSummaries for the motivation.
func (s *server) WatchStageSummaries(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.WatchStageSummariesRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchStageSummariesResponse],
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
		}, &kargoapi.Stage{}); err != nil {
			return fmt.Errorf("get stage: %w", err)
		}
	}

	want := warehouseNameSet(req.Msg.GetFreightOrigins())

	opts := []libClient.ListOption{libClient.InNamespace(project)}
	if name != "" {
		opts = append(opts, libClient.MatchingFields{"metadata.name": name})
	}
	if rv := req.Msg.GetResourceVersion(); rv != "" {
		opts = append(opts, &libClient.ListOptions{
			Raw: &metav1.ListOptions{ResourceVersion: rv},
		})
	}

	w, err := s.client.Watch(ctx, &kargoapi.StageList{}, opts...)
	if err != nil {
		return fmt.Errorf("watch stage summaries: %w", err)
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
			if len(want) > 0 && !stageMatchesAnyWarehouse(stage, want) {
				continue
			}
			if err := stream.Send(&svcv1alpha1.WatchStageSummariesResponse{
				StageSummary: stageToSummary(stage),
				Type:         string(e.Type),
			}); err != nil {
				return fmt.Errorf("send response: %w", err)
			}
		}
	}
}
