package clusterpromotiontask

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libWebhook "github.com/akuity/kargo/pkg/webhook/kubernetes"
)

var clusterPromotionTaskGroupKind = schema.GroupKind{
	Group: kargoapi.GroupVersion.Group,
	Kind:  "ClusterPromotionTask",
}

type webhook struct{}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kargoapi.ClusterPromotionTask{}).
		WithValidator(&webhook{}).
		Complete()
}

func (w *webhook) ValidateCreate(
	_ context.Context,
	obj runtime.Object,
) (admission.Warnings, error) {
	task := obj.(*kargoapi.ClusterPromotionTask) // nolint: forcetypeassert
	if errs := w.validateSpec(field.NewPath("spec"), task.Spec); len(errs) > 0 {
		return nil, apierrors.NewInvalid(
			clusterPromotionTaskGroupKind,
			task.Name,
			errs,
		)
	}
	return nil, nil
}

func (w *webhook) ValidateUpdate(
	_ context.Context,
	_ runtime.Object,
	newObj runtime.Object,
) (admission.Warnings, error) {
	task := newObj.(*kargoapi.ClusterPromotionTask) // nolint: forcetypeassert
	if errs := w.validateSpec(field.NewPath("spec"), task.Spec); len(errs) > 0 {
		return nil, apierrors.NewInvalid(
			clusterPromotionTaskGroupKind,
			task.Name,
			errs,
		)
	}
	return nil, nil
}

func (w *webhook) ValidateDelete(
	context.Context,
	runtime.Object,
) (admission.Warnings, error) {
	// No-op
	return nil, nil
}

func (w *webhook) validateSpec(
	f *field.Path,
	spec kargoapi.PromotionTaskSpec,
) field.ErrorList {
	return libWebhook.ValidatePromotionSteps(f.Child("steps"), spec.Steps)
}
