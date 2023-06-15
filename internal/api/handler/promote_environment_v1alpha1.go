package handler

import (
	"context"
	"fmt"

	"github.com/bufbuild/connect-go"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type PromoteEnvironmentV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.PromoteEnvironmentRequest],
) (*connect.Response[svcv1alpha1.PromoteEnvironmentResponse], error)

func PromoteEnvironmentV1Alpha1(
	kc client.Client,
) PromoteEnvironmentV1Alpha1Func {
	return func(
		ctx context.Context,
		req *connect.Request[svcv1alpha1.PromoteEnvironmentRequest],
	) (*connect.Response[svcv1alpha1.PromoteEnvironmentResponse], error) {
		if req.Msg.GetState() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("state should not be empty"))
		}

		var env kubev1alpha1.Environment
		if err := kc.Get(ctx, client.ObjectKey{
			Namespace: req.Msg.GetProject(),
			Name:      req.Msg.GetName(),
		}, &env); err != nil {
			if kubeerr.IsNotFound(err) {
				return nil, connect.NewError(connect.CodeNotFound,
					fmt.Errorf("environment %q not found", req.Msg.GetName()))
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		stateExists := false
		for _, state := range env.Status.AvailableStates {
			if req.Msg.GetState() == state.ID {
				stateExists = true
				break
			}
		}
		if !stateExists {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("state %q not found in environment %q", req.Msg.GetState(), env.Name))
		}

		promotion := &kubev1alpha1.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-", req.Msg.GetName()),
				Namespace:    req.Msg.GetProject(),
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         kubev1alpha1.GroupVersion.String(),
						Kind:               "Environment",
						Name:               env.Name,
						UID:                env.UID,
						BlockOwnerDeletion: pointer.Bool(true),
					},
				},
			},
			Spec: &kubev1alpha1.PromotionSpec{
				Environment: env.Name,
				State:       req.Msg.GetState(),
			},
		}
		if err := kc.Create(ctx, promotion); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.PromoteEnvironmentResponse{
			Promotion: toPromotionProto(*promotion),
		}), nil
	}
}
