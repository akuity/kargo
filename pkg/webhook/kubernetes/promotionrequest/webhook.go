package promotionrequest

import (
	"context"
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	libWebhook "github.com/akuity/kargo/pkg/webhook/kubernetes"
)

var promotionRequestGroupKind = schema.GroupKind{
	Group: kargoapi.GroupVersion.Group,
	Kind:  "PromotionRequest",
}

type webhook struct {
	client client.Client
}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	w := &webhook{
		client: mgr.GetClient(),
	}
	return ctrl.NewWebhookManagedBy(mgr, &kargoapi.PromotionRequest{}).
		WithValidator(w).
		Complete()
}

func (w *webhook) ValidateCreate(
	ctx context.Context,
	promotionRequest *kargoapi.PromotionRequest,
) (admission.Warnings, error) {
	var errs field.ErrorList
	if err := libWebhook.ValidateProject(ctx, w.client, promotionRequest); err != nil {
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

	specErrs, internalErr := w.validateSpec(
		ctx,
		field.NewPath("spec"),
		promotionRequest,
	)
	if internalErr != nil {
		return nil, apierrors.NewInternalError(internalErr)
	}
	if errs = append(errs, specErrs...); len(errs) > 0 {
		return nil, apierrors.NewInvalid(
			promotionRequestGroupKind,
			promotionRequest.Name,
			errs,
		)
	}
	return nil, nil
}

func (w *webhook) ValidateUpdate(
	ctx context.Context,
	_ *kargoapi.PromotionRequest,
	promotionRequest *kargoapi.PromotionRequest,
) (admission.Warnings, error) {
	// spec.stage and spec.freight cannot have changed -- the schema's transition
	// rules reject that before this runs -- but spec.targets can, so the same
	// checks apply to an update as to a create.
	errs, internalErr := w.validateSpec(
		ctx,
		field.NewPath("spec"),
		promotionRequest,
	)
	if internalErr != nil {
		return nil, apierrors.NewInternalError(internalErr)
	}
	if len(errs) > 0 {
		return nil, apierrors.NewInvalid(
			promotionRequestGroupKind,
			promotionRequest.Name,
			errs,
		)
	}
	return nil, nil
}

func (w *webhook) ValidateDelete(
	context.Context,
	*kargoapi.PromotionRequest,
) (admission.Warnings, error) {
	// No-op
	return nil, nil
}

// validateSpec returns a list of field errors describing everything wrong with
// the PromotionRequest's spec, or a non-nil error if a check could not be
// carried out at all. The two are distinct: the former rejects the object, the
// latter fails the admission request.
func (w *webhook) validateSpec(
	ctx context.Context,
	f *field.Path,
	promotionRequest *kargoapi.PromotionRequest,
) (field.ErrorList, error) {
	var errs field.ErrorList

	errs = append(errs, validateTargetsUnique(
		f.Child("targets"),
		promotionRequest.Spec.Targets,
	)...)

	stageErrs, err := w.validateStage(ctx, f.Child("stage"), promotionRequest)
	if err != nil {
		return nil, err
	}
	errs = append(errs, stageErrs...)

	targetErrs, err := w.validateTargetsExist(
		ctx,
		f.Child("targets"),
		promotionRequest,
	)
	if err != nil {
		return nil, err
	}
	return append(errs, targetErrs...), nil
}

// validateTargetsUnique enforces the uniqueness of Target names.
//
// The schema cannot do this: spec.targets is an atomic list rather than a
// list-map precisely to avoid the per-item ownership tracking a list-map
// records in managedFields, which roughly doubles the storage cost of every
// entry. A CEL rule is no better, since expressing uniqueness costs O(n^2)
// over the very lists that motivated the atomic list in the first place.
func validateTargetsUnique(
	f *field.Path,
	targets []kargoapi.PromotionRequestTarget,
) field.ErrorList {
	var errs field.ErrorList
	seen := make(map[string]struct{}, len(targets))
	for i, target := range targets {
		if _, duplicate := seen[target.Name]; duplicate {
			errs = append(errs, field.Duplicate(f.Index(i).Child("name"), target.Name))
			continue
		}
		seen[target.Name] = struct{}{}
	}
	return errs
}

// validateStage confirms that the named Stage exists and governs Targets. A
// PromotionRequest promotes Freight to a Stage's Targets, so a classic Stage --
// one that promotes to itself -- has nothing for a PromotionRequest to do.
func (w *webhook) validateStage(
	ctx context.Context,
	f *field.Path,
	promotionRequest *kargoapi.PromotionRequest,
) (field.ErrorList, error) {
	stage, err := api.GetStage(
		ctx,
		w.client,
		types.NamespacedName{
			Namespace: promotionRequest.Namespace,
			Name:      promotionRequest.Spec.Stage,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"error getting Stage %q in namespace %q: %w",
			promotionRequest.Spec.Stage, promotionRequest.Namespace, err,
		)
	}
	if stage == nil {
		return field.ErrorList{
			field.Invalid(
				f,
				promotionRequest.Spec.Stage,
				fmt.Sprintf(
					"Stage %q not found in namespace %q",
					promotionRequest.Spec.Stage, promotionRequest.Namespace,
				),
			),
		}, nil
	}
	if stage.Spec.Targets == nil {
		return field.ErrorList{
			field.Invalid(
				f,
				promotionRequest.Spec.Stage,
				fmt.Sprintf(
					"Stage %q does not select Targets; it is promoted to with a "+
						"Promotion rather than a PromotionRequest",
					promotionRequest.Spec.Stage,
				),
			),
		}, nil
	}
	return nil, nil
}

// validateTargetsExist confirms that every named Target exists in the
// PromotionRequest's own Project.
//
// It deliberately does not check that each Target still matches the Stage's
// selectors. spec.targets is a snapshot of what the Stage governed when the
// request was created, and a Target's labels changing afterwards must not
// retroactively invalidate a request that is already in flight.
func (w *webhook) validateTargetsExist(
	ctx context.Context,
	f *field.Path,
	promotionRequest *kargoapi.PromotionRequest,
) (field.ErrorList, error) {
	if len(promotionRequest.Spec.Targets) == 0 {
		return nil, nil
	}

	// One List beats one Get per Target: a fleet PromotionRequest may name
	// thousands, and this runs on every admission.
	list := kargoapi.TargetList{}
	if err := w.client.List(
		ctx,
		&list,
		client.InNamespace(promotionRequest.Namespace),
	); err != nil {
		return nil, fmt.Errorf(
			"error listing Targets in namespace %q: %w",
			promotionRequest.Namespace, err,
		)
	}
	existing := make(map[string]struct{}, len(list.Items))
	for _, target := range list.Items {
		existing[target.Name] = struct{}{}
	}

	var errs field.ErrorList
	for i, target := range promotionRequest.Spec.Targets {
		if _, ok := existing[target.Name]; !ok {
			errs = append(errs, field.Invalid(
				f.Index(i).Child("name"),
				target.Name,
				fmt.Sprintf(
					"Target %q not found in namespace %q",
					target.Name, promotionRequest.Namespace,
				),
			))
		}
	}
	return errs, nil
}
