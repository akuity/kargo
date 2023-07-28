package handler

import (
	"context"

	"github.com/bufbuild/connect-go"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type CreatePromotionPolicyV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.CreatePromotionPolicyRequest],
) (*connect.Response[svcv1alpha1.CreatePromotionPolicyResponse], error)

func CreatePromotionPolicyV1Alpha1(
	kc client.Client,
) CreatePromotionPolicyV1Alpha1Func {
	validateProject := newProjectValidator(kc)
	return func(
		ctx context.Context,
		req *connect.Request[svcv1alpha1.CreatePromotionPolicyRequest],
	) (*connect.Response[svcv1alpha1.CreatePromotionPolicyResponse], error) {
		var policy v1alpha1.PromotionPolicy
		switch {
		case req.Msg.GetYaml() != "":
			if err := yaml.Unmarshal([]byte(req.Msg.GetYaml()), &policy); err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "invalid yaml"))
			}
		case req.Msg.GetTyped() != nil:
			if req.Msg.GetTyped().GetProject() == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
			}
			if req.Msg.GetTyped().GetName() == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name should not be empty"))
			}
			policy = v1alpha1.PromotionPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: req.Msg.GetTyped().GetProject(),
					Name:      req.Msg.GetTyped().GetName(),
				},
				Stage:               req.Msg.GetTyped().GetStage(),
				EnableAutoPromotion: req.Msg.GetTyped().GetEnableAutoPromotion(),
			}
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("promotion_policy should not be empty"))
		}

		if err := validateProject(ctx, policy.GetNamespace()); err != nil {
			return nil, err
		}
		if err := kc.Create(ctx, &policy); err != nil {
			if kubeerr.IsAlreadyExists(err) {
				return nil, connect.NewError(connect.CodeAlreadyExists, err)
			}
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "create promotion policy"))
		}
		return connect.NewResponse(&svcv1alpha1.CreatePromotionPolicyResponse{
			PromotionPolicy: typesv1alpha1.ToPromotionPolicyProto(policy),
		}), nil
	}
}
