package api

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
	corev1 "k8s.io/api/core/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) CreateProject(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateProjectRequest],
) (*connect.Response[svcv1alpha1.CreateProjectResponse], error) {
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name should not be empty"))
	}

	var existingNs corev1.Namespace
	if err := s.client.Get(ctx, client.ObjectKey{Name: name}, &existingNs); err == nil || !kubeerr.IsNotFound(err) {
		if err != nil {
			return nil, errors.Wrap(err, "get namespace")
		}
		if existingNs.GetLabels()[kargoapi.LabelProjectKey] == kargoapi.LabelTrueValue {
			return nil, connect.NewError(connect.CodeAlreadyExists,
				errors.Errorf("project %q already exists", name))
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.Errorf("non-project namespace %q already exists", name))
	}

	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				kargoapi.LabelProjectKey: kargoapi.LabelTrueValue,
			},
			Name: name,
		},
	}
	if err := s.client.Create(ctx, &ns); err != nil {
		return nil, errors.Wrap(err, "create namespace")
	}
	return connect.NewResponse(&svcv1alpha1.CreateProjectResponse{
		Project: &svcv1alpha1.Project{
			Name:       ns.Name,
			CreateTime: timestamppb.New(ns.CreationTimestamp.Time),
		},
	}), nil
}
