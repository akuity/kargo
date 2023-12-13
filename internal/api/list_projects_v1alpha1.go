package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) ListProjects(
	ctx context.Context,
	_ *connect.Request[svcv1alpha1.ListProjectsRequest],
) (*connect.Response[svcv1alpha1.ListProjectsResponse], error) {
	// Only list namespaces which are labeled as Kargo projects
	selector := labels.Set{kargoapi.LabelProjectKey: kargoapi.LabelTrueValue}.AsSelector()
	nsList := &corev1.NamespaceList{}
	if err := s.client.List(ctx, nsList, client.MatchingLabelsSelector{Selector: selector}); err != nil {
		return nil, connect.NewError(getCodeFromError(err), errors.Wrap(err, "list projects"))
	}

	projects := make([]*svcv1alpha1.Project, len(nsList.Items))
	for i, ns := range nsList.Items {
		projects[i] = &svcv1alpha1.Project{
			Name:       ns.Name,
			CreateTime: timestamppb.New(ns.CreationTimestamp.Time),
		}
	}
	return connect.NewResponse(&svcv1alpha1.ListProjectsResponse{
		Projects: projects,
	}), nil
}
