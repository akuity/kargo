package handler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/bufbuild/connect-go"
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
		namespaceList := &corev1.NamespaceList{}
		if err := kc.List(ctx, namespaceList); err != nil {
			return nil, err
		}

		projects := make([]*svcv1alpha1.Project, 0, len(namespaceList.Items))

		fmt.Println("FOO")

		for _, namespace := range namespaceList.Items {
			fmt.Println(namespace.Name)
			projects = append(projects, &svcv1alpha1.Project{
				Name: namespace.Name,
			})
		}
		return connect.NewResponse(&svcv1alpha1.ListProjectsResponse{
			Projects: projects,
		}), nil
	}
}
