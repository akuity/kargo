package promotionpolicy

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
	libWebhook "github.com/akuity/kargo/internal/webhook"
)

var (
	promotionPolicyGroupKind = schema.GroupKind{
		Group: kargoapi.GroupVersion.Group,
		Kind:  "PromotionPolicy",
	}
	promotionPolicyGroupResource = schema.GroupResource{
		Group:    kargoapi.GroupVersion.Group,
		Resource: "PromotionPolicy",
	}
)

type webhook struct {
	client client.Client

	// The following behaviors are overridable for testing purposes:

	validateProjectFn func(
		context.Context,
		client.Client,
		schema.GroupKind,
		client.Object,
	) error

	validateStageUniquenessFn func(
		context.Context,
		*kargoapi.PromotionPolicy,
	) error

	listPromotionPoliciesFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error
}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	w := newWebhook(mgr.GetClient())
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kargoapi.PromotionPolicy{}).
		WithValidator(w).
		Complete()
}

func newWebhook(kubeClient client.Client) *webhook {
	w := &webhook{
		client: kubeClient,
	}
	w.validateProjectFn = libWebhook.ValidateProject
	w.validateStageUniquenessFn = w.validateStageUniqueness
	w.listPromotionPoliciesFn = kubeClient.List
	return w
}

func (w *webhook) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) error {
	policy := obj.(*kargoapi.PromotionPolicy) // nolint: forcetypeassert
	if err := w.validateProjectFn(
		ctx,
		w.client,
		promotionPolicyGroupKind,
		policy,
	); err != nil {
		return err
	}
	return w.validateStageUniquenessFn(ctx, policy)
}

func (w *webhook) ValidateUpdate(
	ctx context.Context,
	_ runtime.Object,
	newObj runtime.Object,
) error {
	policy := newObj.(*kargoapi.PromotionPolicy) // nolint: forcetypeassert
	return w.validateStageUniquenessFn(ctx, policy)
}

func (w *webhook) ValidateDelete(context.Context, runtime.Object) error {
	// No-op
	return nil
}

func (w *webhook) validateStageUniqueness(
	ctx context.Context,
	policy *kargoapi.PromotionPolicy,
) error {
	var list kargoapi.PromotionPolicyList
	if err := w.listPromotionPoliciesFn(
		ctx,
		&list,
		client.InNamespace(policy.GetNamespace()),
		client.MatchingFields{
			kubeclient.PromotionPoliciesByStageIndexField: policy.Stage,
		},
	); err != nil {
		return apierrors.NewInternalError(
			errors.Wrap(err, "list promotion policies"),
		)
	}
	for _, ep := range list.Items {
		if policy.Name != ep.Name {
			return apierrors.NewConflict(
				promotionPolicyGroupResource,
				policy.GetName(),
				fmt.Errorf(
					"policy for stage %q is already exists: %s",
					policy.Stage,
					ep.GetName(),
				),
			)
		}
	}
	return nil
}
