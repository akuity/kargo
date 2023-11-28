package freight

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
	}
}

func (w *webhook) Default(_ context.Context, obj runtime.Object) error {
	freight := obj.(*kargoapi.Freight) // nolint: forcetypeassert
	// Re-calculate ID in case it wasn't set correctly to begin with -- possible
	// if/when we allow users to create their own Freight.
	freight.UpdateID()
	// TODO: For now, we'll force Name to be the same as ID, but be can change
	// this later if/when we allow users to create their own Freight.
	freight.Name = freight.ID
	return nil
}

func (w *webhook) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) error {
	freight := obj.(*kargoapi.Freight) // nolint: forcetypeassert
	if err :=
		w.validateProjectFn(ctx, w.client, freightGroupKind, freight); err != nil {
		return err
	}
	if len(freight.Commits) == 0 &&
		len(freight.Images) == 0 &&
		len(freight.Charts) == 0 {
		return apierrors.NewInvalid(
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
	return nil
}

func (w *webhook) ValidateUpdate(
	_ context.Context,
	oldObj runtime.Object,
	newObj runtime.Object,
) error {
	freight := newObj.(*kargoapi.Freight) // nolint: forcetypeassert
	// Freight is meant to be immutable. We only need to compare IDs because IDs
	// are fingerprints that are deterministically derived from the artifacts
	// referenced by the Freight.
	if freight.ID != (oldObj.(*kargoapi.Freight)).ID { // nolint: forcetypeassert
		return apierrors.NewInvalid(
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
	return nil
}

func (w *webhook) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	freight := obj.(*kargoapi.Freight) // nolint: forcetypeassert

	// Check if the given freight is used by any stages.
	var list kargoapi.StageList
	if err := w.client.List(
		ctx,
		&list,
		client.InNamespace(freight.GetNamespace()),
		client.MatchingFields{
			kubeclient.StagesByFreightIndexField: freight.ID,
		},
	); err != nil {
		return errors.Wrap(err, "list stages")
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
		return apierrors.NewForbidden(freightGroupResource, freight.Name, err)
	}
	return nil
}
