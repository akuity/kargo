package api

import (
	"context"
	"fmt"
	"math"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) ListImages(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListImagesRequest],
) (*connect.Response[svcv1alpha1.ListImagesResponse], error) {
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

	stages := make([]*kargoapi.Stage, len(list.Items))
	images := make(map[string]*svcv1alpha1.TagMap)

	for idx, stage := range list.Items {
		stages[idx] = &list.Items[idx]

		for i, freightGroup := range stage.Status.FreightHistory {
			if i > math.MaxInt32 {
				return nil, fmt.Errorf("index %d exceeds maximum value for int32", i)
			}
			safeI := int32(math.Min(float64(i), math.MaxInt32))

			for _, freight := range freightGroup.Freight {
				for _, image := range freight.Images {
					repo, ok := images[image.RepoURL]
					if !ok || repo == nil {
						repo = &svcv1alpha1.TagMap{}
						images[image.RepoURL] = repo
					}

					if repo.Tags == nil {
						repo.Tags = make(map[string]*svcv1alpha1.ImageStageMap)
					}

					stagemap, ok := repo.Tags[image.Tag]
					if !ok || stagemap == nil {
						repo.Tags[image.Tag] = &svcv1alpha1.ImageStageMap{}
						stagemap = repo.Tags[image.Tag]
					}

					if stagemap.Stages == nil {
						stagemap.Stages = make(map[string]int32)
					}

					if _, ok := stagemap.Stages[stage.Name]; !ok {
						stagemap.Stages[stage.Name] = safeI
					}
				}
			}
		}
	}

	return connect.NewResponse(&svcv1alpha1.ListImagesResponse{
		Images: images,
	}), nil
}
