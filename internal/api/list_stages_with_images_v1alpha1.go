package api

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"connectrpc.com/connect"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) ListStagesWithImages(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListStagesRequest],
) (*connect.Response[svcv1alpha1.ListStagesWithImagesResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	var list kargoapi.StageList
	if err := s.client.List(ctx, &list, client.InNamespace(project)); err != nil {
		return nil, fmt.Errorf("list stages: %w", err)
	}

	images := make(map[string]*svcv1alpha1.Image)
	stages := make([]*svcv1alpha1.StageWithImages, len(list.Items))
	for _, stage := range list.Items {
		for i, freightGroup := range stage.Status.FreightHistory {
			for _, freight := range freightGroup.Freight {
				for _, image := range freight.Images {
					repo, ok := images[image.RepoURL]
					if !ok || repo == nil {
						repo := &svcv1alpha1.Image{}
						images[image.RepoURL] = repo
					}

					_, ok = repo.Tags[image.Tag]
					if !ok {
						repo.Tags[image.Tag] = int32(i)
					}
				}
			}
		}
	}

	return connect.NewResponse(&svcv1alpha1.ListStagesWithImagesResponse{
		Stages: stages,
		Images: images,
	}), nil
}
