package handler

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type DeleteProjectV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.DeleteProjectRequest],
) (*connect.Response[svcv1alpha1.DeleteProjectResponse], error)

func DeleteProjectV1Alpha1(
	kc client.Client,
) DeleteProjectV1Alpha1Func {
	return func(
		ctx context.Context,
		req *connect.Request[svcv1alpha1.DeleteProjectRequest],
	) (*connect.Response[svcv1alpha1.DeleteProjectResponse], error) {
		name := strings.TrimSpace(req.Msg.GetName())
		if name == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name should not be empty"))
		}

		var ns corev1.Namespace
		if err := kc.Get(ctx, client.ObjectKey{Name: name}, &ns); err != nil {
			if kubeerr.IsNotFound(err) {
				return nil, connect.NewError(connect.CodeNotFound,
					errors.Errorf("project %q not found", name))
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if ns.GetLabels()[v1alpha1.LabelProjectKey] != v1alpha1.LabelTrueValue {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				errors.Errorf("namespace %q is not a project", ns.GetName()))
		}
		if err := kc.Delete(ctx, &ns); err != nil && !kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.DeleteProjectResponse{
			/* explicitly empty */
		}), nil
	}
}
