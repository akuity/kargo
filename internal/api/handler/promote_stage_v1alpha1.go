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

type PromoteStageV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.PromoteStageRequest],
) (*connect.Response[svcv1alpha1.PromoteStageResponse], error)

func PromoteStageV1Alpha1(
	kc client.Client,
) PromoteStageV1Alpha1Func {
	return func(
		ctx context.Context,
		req *connect.Request[svcv1alpha1.PromoteStageRequest],
	) (*connect.Response[svcv1alpha1.PromoteStageResponse], error) {
		if req.Msg.GetState() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("state should not be empty"))
		}

		var stage kubev1alpha1.Stage
		if err := kc.Get(ctx, client.ObjectKey{
			Namespace: req.Msg.GetProject(),
			Name:      req.Msg.GetName(),
		}, &stage); err != nil {
			if kubeerr.IsNotFound(err) {
				return nil, connect.NewError(connect.CodeNotFound,
					fmt.Errorf("stage %q not found", req.Msg.GetName()))
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		stateExists := false
		for _, state := range stage.Status.AvailableStates {
			if req.Msg.GetState() == state.ID {
				stateExists = true
				break
			}
		}
		if !stateExists {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("state %q not found in Stage %q", req.Msg.GetState(), stage.Name))
		}

		promotion := &kubev1alpha1.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-", req.Msg.GetName()),
				Namespace:    req.Msg.GetProject(),
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         kubev1alpha1.GroupVersion.String(),
						Kind:               "Stage",
						Name:               stage.Name,
						UID:                stage.UID,
						BlockOwnerDeletion: pointer.Bool(true),
					},
				},
			},
			Spec: &kubev1alpha1.PromotionSpec{
				Stage: stage.Name,
				State: req.Msg.GetState(),
			},
		}
		if err := kc.Create(ctx, promotion); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.PromoteStageResponse{
			Promotion: toPromotionProto(*promotion),
		}), nil
	}
}
