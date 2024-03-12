package freight

import (
	"context"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
	libWebhook "github.com/akuity/kargo/internal/webhook"
)

var (
	freightGroupKind = schema.GroupKind{
		Group: kargoapi.GroupVersion.Group,
		Kind:  "Freight",
	}
	freightGroupResource = schema.GroupResource{
		Group:    kargoapi.GroupVersion.Group,
		Resource: "freights",
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

	listFreightFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error

	listStagesFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error
}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	w := newWebhook(mgr.GetClient())
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kargoapi.Freight{}).
		WithValidator(w).
		WithDefaulter(w).
		Complete()
}

func newWebhook(kubeClient client.Client) *webhook {
	return &webhook{
		client:            kubeClient,
		validateProjectFn: libWebhook.ValidateProject,
		listFreightFn:     kubeClient.List,
		listStagesFn:      kubeClient.List,
	}
}

func (w *webhook) Default(_ context.Context, obj runtime.Object) error {
	freight := obj.(*kargoapi.Freight) // nolint: forcetypeassert
	// Re-calculate ID in case it wasn't set correctly to begin with -- possible
	// if/when we allow users to create their own Freight.
	freight.Name = freight.GenerateID()
	return nil
}

func (w *webhook) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) (admission.Warnings, error) {
	freight := obj.(*kargoapi.Freight) // nolint: forcetypeassert
	if err :=
		w.validateProjectFn(ctx, w.client, freightGroupKind, freight); err != nil {
		return nil, err
	}

	if freight.ObjectMeta.Labels != nil &&
		freight.ObjectMeta.Labels[kargoapi.AliasLabelKey] != "" {
		alias := freight.ObjectMeta.Labels[kargoapi.AliasLabelKey]
		freightList := kargoapi.FreightList{}
		if err := w.listFreightFn(
			ctx,
			&freightList,
			client.InNamespace(freight.Namespace),
			client.MatchingLabels{kargoapi.AliasLabelKey: alias},
		); err != nil {
			return nil, apierrors.NewInternalError(err)
		}
		if len(freightList.Items) > 0 {
			return nil, apierrors.NewConflict(
				freightGroupResource,
				freight.Name,
				fmt.Errorf(
					"alias %q already used by another piece of Freight in namespace %q",
					alias,
					freight.Namespace,
				),
			)
		}
	}

	if len(freight.Commits) == 0 &&
		len(freight.Images) == 0 &&
		len(freight.Charts) == 0 {
		return nil, apierrors.NewInvalid(
			freightGroupKind,
			freight.Name,
			field.ErrorList{
				field.Invalid(
					field.NewPath(""),
					freight,
					"freight must contain at least one commit, image, or chart",
				),
			},
		)
	}
	return nil, nil
}

func (w *webhook) ValidateUpdate(
	ctx context.Context,
	_ runtime.Object,
	newObj runtime.Object,
) (admission.Warnings, error) {
	freight := newObj.(*kargoapi.Freight) // nolint: forcetypeassert

	if freight.ObjectMeta.Labels != nil &&
		freight.ObjectMeta.Labels[kargoapi.AliasLabelKey] != "" {
		alias := freight.ObjectMeta.Labels[kargoapi.AliasLabelKey]
		freightList := kargoapi.FreightList{}
		if err := w.listFreightFn(
			ctx,
			&freightList,
			client.InNamespace(freight.Namespace),
			client.MatchingLabels{kargoapi.AliasLabelKey: alias},
		); err != nil {
			return nil, apierrors.NewInternalError(err)
		}
		if len(freightList.Items) > 1 ||
			(len(freightList.Items) == 1 && freightList.Items[0].Name != freight.Name) {
			return nil, apierrors.NewConflict(
				freightGroupResource,
				freight.Name,
				fmt.Errorf(
					"alias %q already used by another piece of Freight in namespace %q",
					alias,
					freight.Namespace,
				),
			)
		}
	}

	// Freight is meant to be immutable. We only need to compare the Name to a
	// newly generated ID because these are both fingerprints that are
	// deterministically derived from the artifacts referenced by the Freight.
	if freight.Name != freight.GenerateID() { // nolint: forcetypeassert
		return nil, apierrors.NewInvalid(
			freightGroupKind,
			freight.Name,
			field.ErrorList{
				field.Invalid(
					field.NewPath(""),
					freight,
					"freight is immutable",
				),
			},
		)
	}
	return nil, nil
}

func (w *webhook) ValidateDelete(
	ctx context.Context,
	obj runtime.Object,
) (admission.Warnings, error) {
	freight := obj.(*kargoapi.Freight) // nolint: forcetypeassert

	// Check if the given freight is used by any stages.
	var list kargoapi.StageList
	if err := w.listStagesFn(
		ctx,
		&list,
		client.InNamespace(freight.GetNamespace()),
		client.MatchingFields{
			kubeclient.StagesByFreightIndexField: freight.Name,
		},
	); err != nil {
		return nil, fmt.Errorf("list stages: %w", err)
	}
	if len(list.Items) > 0 {
		stages := make([]string, len(list.Items))
		for i, stage := range list.Items {
			stages[i] = fmt.Sprintf("%q", stage.Name)
		}
		err := fmt.Errorf(
			"freight is in-use by stages (%s)",
			strings.Join(stages, ", "),
		)
		return nil, apierrors.NewForbidden(freightGroupResource, freight.Name, err)
	}
	return nil, nil
}
