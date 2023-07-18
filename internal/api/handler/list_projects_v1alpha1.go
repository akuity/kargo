package handler

import (
	"context"

	"github.com/bufbuild/connect-go"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/api/v1alpha1"
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
		// Only list namespaces which are labeled as Kargo projects
		selector := labels.Set{v1alpha1.LabelProjectKey: v1alpha1.LabelTrueValue}.AsSelector()
		nsList := &corev1.NamespaceList{}
		if err := kc.List(ctx, nsList, client.MatchingLabelsSelector{Selector: selector}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		var projects []*svcv1alpha1.Project
		for _, ns := range nsList.Items {
			projects = append(projects, &svcv1alpha1.Project{
				Name: ns.Name,
			})
		}

		return connect.NewResponse(&svcv1alpha1.ListProjectsResponse{
			Projects: projects,
		}), nil
	}
}
