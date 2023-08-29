package handler

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

	"github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type CreateProjectV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.CreateProjectRequest],
) (*connect.Response[svcv1alpha1.CreateProjectResponse], error)

func CreateProjectV1Alpha1(
	kc client.Client,
) CreateProjectV1Alpha1Func {
	return func(
		ctx context.Context,
		req *connect.Request[svcv1alpha1.CreateProjectRequest],
	) (*connect.Response[svcv1alpha1.CreateProjectResponse], error) {
		name := strings.TrimSpace(req.Msg.GetName())
		if name == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name should not be empty"))
		}

		var existingNs corev1.Namespace
		if err := kc.Get(ctx, client.ObjectKey{Name: name}, &existingNs); err == nil || !kubeerr.IsNotFound(err) {
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal,
					errors.Wrap(err, "get existing namespace"))
			}
			if existingNs.GetLabels()[v1alpha1.LabelProjectKey] == v1alpha1.LabelTrueValue {
				return nil, connect.NewError(connect.CodeAlreadyExists,
					errors.Errorf("project %q already exists", name))
			}
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				errors.Errorf("non-project namespace %q already exists", name))
		}

		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					v1alpha1.LabelProjectKey: v1alpha1.LabelTrueValue,
				},
				Name: name,
			},
		}
		if err := kc.Create(ctx, &ns); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.CreateProjectResponse{
			Project: &svcv1alpha1.Project{
				Name:       ns.Name,
				CreateTime: timestamppb.New(ns.CreationTimestamp.Time),
			},
		}), nil
	}
}
