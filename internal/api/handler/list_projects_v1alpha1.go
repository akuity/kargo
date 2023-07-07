package handler

import (
	"context"

	"github.com/bufbuild/connect-go"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type ListProjectsV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.ListProjectsRequest],
) (*connect.Response[svcv1alpha1.ListProjectsResponse], error)

func ListProjectsV1Alpha1(
	kc client.Client,
) ListProjectsV1Alpha1Func {
	return func(
		ctx context.Context,
		req *connect.Request[svcv1alpha1.ListProjectsRequest],
	) (*connect.Response[svcv1alpha1.ListProjectsResponse], error) {

		// Only list projects which contain an Environment
		var list kubev1alpha1.EnvironmentList
		if err := kc.List(ctx, &list); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		var projectMap = make(map[string]bool)
		var projects []*svcv1alpha1.Project
		for _, env := range list.Items {
			if _, ok := projectMap[env.Namespace]; ok {
				continue
			}
			projectMap[env.Namespace] = true
			projects = append(projects, &svcv1alpha1.Project{
				Name: env.Namespace,
			})
		}

		return connect.NewResponse(&svcv1alpha1.ListProjectsResponse{
			Projects: projects,
		}), nil
	}
}
