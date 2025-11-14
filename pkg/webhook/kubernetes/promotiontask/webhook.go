package promotiontask

import (
	"context"
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libWebhook "github.com/akuity/kargo/pkg/webhook/kubernetes"
)

var promotionTaskGroupKind = schema.GroupKind{
	Group: kargoapi.GroupVersion.Group,
	Kind:  "PromotionTask",
}

type webhook struct {
	client client.Client
}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	w := &webhook{
		client: mgr.GetClient(),
	}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kargoapi.PromotionTask{}).
		WithValidator(w).
		Complete()
}

func (w *webhook) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) (admission.Warnings, error) {
	task := obj.(*kargoapi.PromotionTask) // nolint: forcetypeassert
	var errs field.ErrorList
	if err := libWebhook.ValidateProject(ctx, w.client, task); err != nil {
		var statusErr *apierrors.StatusError
		if ok := errors.As(err, &statusErr); ok {
			return nil, statusErr
		}
		var fieldErr *field.Error
		if ok := errors.As(err, &fieldErr); !ok {
			return nil, apierrors.NewInternalError(err)
		}
		errs = append(errs, fieldErr)
	}
	if errs = append(
		errs,
		w.validateSpec(field.NewPath("spec"), task.Spec)...,
	); len(errs) > 0 {
		return nil, apierrors.NewInvalid(promotionTaskGroupKind, task.Name, errs)
	}
	return nil, nil
}

func (w *webhook) ValidateUpdate(
	_ context.Context,
	_ runtime.Object,
	newObj runtime.Object,
) (admission.Warnings, error) {
	task := newObj.(*kargoapi.PromotionTask) // nolint: forcetypeassert
	if errs := w.validateSpec(field.NewPath("spec"), task.Spec); len(errs) > 0 {
		return nil, apierrors.NewInvalid(promotionTaskGroupKind, task.Name, errs)
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
