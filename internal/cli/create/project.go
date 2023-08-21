package create

import (
	"context"

	"connectrpc.com/connect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

func createProject(
	ctx context.Context,
	kargoSvcCli svcv1alpha1connect.KargoServiceClient,
	name string,
) (runtime.Object, error) {
	resp, err := kargoSvcCli.CreateProject(ctx, connect.NewRequest(&v1alpha1.CreateProjectRequest{
		Name: name,
	}))
	if err != nil {
		return nil, err
	}

	var project unstructured.Unstructured
	project.SetAPIVersion(kubev1alpha1.GroupVersion.String())
	project.SetKind("Project")
	project.SetCreationTimestamp(metav1.NewTime(resp.Msg.GetProject().GetCreateTime().AsTime()))
	project.SetName(resp.Msg.GetProject().GetName())
	return &project, nil
}
