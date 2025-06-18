package clusterconfig

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/webhook/kubernetes/external"
)

var clusterConfigGroupKind = schema.GroupKind{
	Group: kargoapi.GroupVersion.Group,
	Kind:  "ClusterConfig",
}

type webhook struct{}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kargoapi.ClusterConfig{}).
		WithValidator(&webhook{}).
		Complete()
}

func (w *webhook) ValidateCreate(
	_ context.Context,
	obj runtime.Object,
) (admission.Warnings, error) {
	clusterCfg := obj.(*kargoapi.ClusterConfig) // nolint: forcetypeassert

	var errs field.ErrorList
	if metaErrs := w.validateObjectMeta(
		field.NewPath("metadata"),
		clusterCfg.ObjectMeta,
	); len(metaErrs) > 0 {
		errs = append(errs, metaErrs...)
	}

	if specErrs := w.validateSpec(
		field.NewPath("spec"),
		clusterCfg.Spec,
	); len(specErrs) > 0 {
		errs = append(errs, specErrs...)
	}

	if len(errs) > 0 {
		return nil, apierrors.NewInvalid(
			clusterConfigGroupKind,
			clusterCfg.Name,
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
	clusterCfg := newObj.(*kargoapi.ClusterConfig) // nolint: forcetypeassert
	if errs := w.validateSpec(field.NewPath("spec"), clusterCfg.Spec); len(errs) > 0 {
		return nil, apierrors.NewInvalid(clusterConfigGroupKind, clusterCfg.Name, errs)
	}
	return nil, nil
}

func (w *webhook) ValidateDelete(
	context.Context,
	runtime.Object,
) (admission.Warnings, error) {
	return nil, nil
}

func (w *webhook) validateObjectMeta(
	f *field.Path,
	meta metav1.ObjectMeta,
) field.ErrorList {
	if meta.Name != api.ClusterConfigName {
		return field.ErrorList{
			field.Invalid(
				f.Child("name"),
				meta.Name,
				fmt.Sprintf("name %q must be %q", meta.Name, api.ClusterConfigName),
			),
		}
	}
	return nil
}

func (w *webhook) validateSpec(
	f *field.Path,
	spec kargoapi.ClusterConfigSpec,
) field.ErrorList {
	var fieldErrs field.ErrorList
	if errs := external.ValidateWebhookReceivers(
		f.Child("webhookReceivers"),
		spec.WebhookReceivers,
	); errs != nil {
		fieldErrs = append(fieldErrs, errs...)
	}
	return fieldErrs
}
