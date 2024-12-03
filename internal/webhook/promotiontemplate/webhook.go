package promotiontemplate

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libWebhook "github.com/akuity/kargo/internal/webhook"
)

var promotionTemplateGroupKind = schema.GroupKind{
	Group: kargoapi.GroupVersion.Group,
	Kind:  "PromotionTemplate",
}

type webhook struct {
	client client.Client
}

func SetupWebhookWithManager(
	mgr ctrl.Manager,
) error {
	w := &webhook{
		client: mgr.GetClient(),
	}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kargoapi.PromotionTemplate{}).
		WithValidator(w).
		Complete()
}

func (w *webhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	template := obj.(*kargoapi.PromotionTemplate) // nolint: forcetypeassert
	if err := libWebhook.ValidateProject(ctx, w.client, promotionTemplateGroupKind, template); err != nil {
		return nil, err
	}
	return nil, nil
}

func (w *webhook) ValidateUpdate(context.Context, runtime.Object, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (w *webhook) ValidateDelete(context.Context, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
