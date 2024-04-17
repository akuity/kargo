package freight

import (
	"context"
	"fmt"
	"strings"

	"github.com/technosophos/moniker"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/record"
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
	client                client.Client
	freightAliasGenerator moniker.Namer

	recorder record.EventRecorder

	// The following behaviors are overridable for testing purposes:

	admissionRequestFromContextFn func(context.Context) (admission.Request, error)

	getAvailableFreightAliasFn func(context.Context) (string, error)

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

	isRequestFromKargoControlplaneFn libWebhook.IsRequestFromKargoControlplaneFn
}

func SetupWebhookWithManager(
	ctx context.Context,
	cfg libWebhook.Config,
	mgr ctrl.Manager,
) error {
	w := newWebhook(
		cfg,
		mgr.GetClient(),
		kubeclient.NewEventRecorder(ctx, mgr.GetScheme(), mgr.GetClient(), "freight-webhook"),
	)
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kargoapi.Freight{}).
		WithValidator(w).
		WithDefaulter(w).
		Complete()
}

func newWebhook(
	cfg libWebhook.Config,
	kubeClient client.Client,
	recorder record.EventRecorder,
) *webhook {
	w := &webhook{
		client:                kubeClient,
		freightAliasGenerator: moniker.New(),
		recorder:              recorder,
	}
	w.admissionRequestFromContextFn = admission.RequestFromContext
	w.getAvailableFreightAliasFn = w.getAvailableFreightAlias
	w.validateProjectFn = libWebhook.ValidateProject
	w.listFreightFn = kubeClient.List
	w.listStagesFn = kubeClient.List
	w.isRequestFromKargoControlplaneFn =
		libWebhook.IsRequestFromKargoControlplane(cfg.ControlplaneUserRegex)
	return w
}

func (w *webhook) Default(ctx context.Context, obj runtime.Object) error {
	freight := obj.(*kargoapi.Freight) // nolint: forcetypeassert
	// Re-calculate ID in case it wasn't set correctly to begin with -- possible
	// if/when we allow users to create their own Freight.
	freight.Name = freight.GenerateID()

	// Sync the convenience alias field with the alias label
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return apierrors.NewInternalError(
			fmt.Errorf("error getting admission request from context: %w", err),
		)
	}
	if freight.Labels == nil {
		freight.Labels = make(map[string]string, 1)
	}
	if freight.Alias != "" {
		// Alias field has a value, so just copy it to the label
		freight.Labels[kargoapi.AliasLabelKey] = freight.Alias
	} else if req.Operation == admissionv1.Create {
		// Alias field is empty and this is a create operation, so generate a new
		// alias and assign it to both the alias field and the label
		var err error
		if freight.Alias, err = w.getAvailableFreightAliasFn(ctx); err != nil {
			return fmt.Errorf("get available freight alias: %w", err)
		}
		freight.Labels[kargoapi.AliasLabelKey] = freight.Alias
	} else {
		// Alias field is empty and this is an update operation, so ensure the
		// label does not exist
		delete(freight.Labels, kargoapi.AliasLabelKey)
	}

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

	freightList := kargoapi.FreightList{}
	if err := w.listFreightFn(
		ctx,
		&freightList,
		client.InNamespace(freight.Namespace),
		client.MatchingLabels{kargoapi.AliasLabelKey: freight.Alias},
	); err != nil {
		return nil, apierrors.NewInternalError(err)
	}
	if len(freightList.Items) > 0 {
		return nil, apierrors.NewConflict(
			freightGroupResource,
			freight.Name,
			fmt.Errorf(
				"alias %q already used by another piece of Freight in namespace %q",
				freight.Alias,
				freight.Namespace,
			),
		)
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
	oldObj runtime.Object,
	newObj runtime.Object,
) (admission.Warnings, error) {
	oldFreight := oldObj.(*kargoapi.Freight) // nolint: forcetypeassert
	newFreight := newObj.(*kargoapi.Freight) // nolint: forcetypeassert

	freightList := kargoapi.FreightList{}
	if err := w.listFreightFn(
		ctx,
		&freightList,
		client.InNamespace(newFreight.Namespace),
		client.MatchingLabels{kargoapi.AliasLabelKey: newFreight.Alias},
	); err != nil {
		return nil, apierrors.NewInternalError(err)
	}
	if len(freightList.Items) > 1 ||
		(len(freightList.Items) == 1 && freightList.Items[0].Name != newFreight.Name) {
		return nil, apierrors.NewConflict(
			freightGroupResource,
			newFreight.Name,
			fmt.Errorf(
				"alias %q already used by another piece of Freight in namespace %q",
				newFreight.Alias,
				newFreight.Namespace,
			),
		)
	}

	// Freight is meant to be immutable. We only need to compare the Name to a
	// newly generated ID because these are both fingerprints that are
	// deterministically derived from the artifacts referenced by the Freight.
	if newFreight.Name != newFreight.GenerateID() || oldFreight.Warehouse != newFreight.Warehouse {
		return nil, apierrors.NewInvalid(
			freightGroupKind,
			oldFreight.Name,
			field.ErrorList{
				field.Invalid(
					field.NewPath(""),
					oldFreight,
					"freight is immutable",
				),
			},
		)
	}

	req, err := w.admissionRequestFromContextFn(ctx)
	if err != nil {
		return nil, fmt.Errorf("get admission request from context: %w", err)
	}
	// Record Freight approved events if the request doesn't come from Kargo controlplane.
	if !w.isRequestFromKargoControlplaneFn(req) {
		for approvedStage := range newFreight.Status.ApprovedFor {
			if _, ok := oldFreight.Status.ApprovedFor[approvedStage]; !ok {
				w.recordFreightApprovedEvent(req, newFreight, approvedStage)
			}
		}
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

func (w *webhook) recordFreightApprovedEvent(
	req admission.Request,
	f *kargoapi.Freight,
	stageName string,
) {
	actor := kargoapi.FormatEventKubernetesUserActor(req.UserInfo)
	w.recorder.AnnotatedEventf(
		f,
		kargoapi.NewFreightApprovedEventAnnotations(actor, f, stageName),
		corev1.EventTypeNormal,
		kargoapi.EventReasonFreightApproved,
		"Freight approved for Stage %q by %q",
		stageName,
		actor,
	)
}
