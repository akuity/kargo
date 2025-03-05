package freight

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/technosophos/moniker"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/indexer"
	libEvent "github.com/akuity/kargo/internal/kubernetes/event"
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

	getWarehouseFn func(context.Context, client.Client, types.NamespacedName) (*kargoapi.Warehouse, error)

	validateFreightArtifactsFn func(*kargoapi.Freight, *kargoapi.Warehouse) error

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
		libEvent.NewRecorder(ctx, mgr.GetScheme(), mgr.GetClient(), "freight-webhook"),
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
	w.getWarehouseFn = api.GetWarehouse
	w.validateFreightArtifactsFn = validateFreightArtifacts
	w.isRequestFromKargoControlplaneFn = libWebhook.IsRequestFromKargoControlplane(cfg.ControlplaneUserRegex)
	return w
}

func (w *webhook) Default(ctx context.Context, obj runtime.Object) error {
	freight := obj.(*kargoapi.Freight) // nolint: forcetypeassert

	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return apierrors.NewInternalError(
			fmt.Errorf("error getting admission request from context: %w", err),
		)
	}
	if req.Operation == admissionv1.Create {
		// Re-calculate ID in case it wasn't set correctly to begin with -- possible
		// when users create their own Freight.
		freight.Name = api.GenerateFreightID(freight)
	}

	// Sync the convenience alias field with the alias label
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
	if err := w.validateProjectFn(ctx, w.client, freightGroupKind, freight); err != nil {
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

	if len(freight.Commits) == 0 && len(freight.Images) == 0 && len(freight.Charts) == 0 {
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

	warehouse, err := w.getWarehouseFn(ctx, w.client, types.NamespacedName{
		Namespace: freight.Namespace,
		Name:      freight.Origin.Name,
	})
	if err != nil {
		return nil, err
	}
	if warehouse == nil {
		return nil, apierrors.NewInvalid(
			freightGroupKind,
			freight.Name,
			field.ErrorList{
				field.Invalid(
					field.NewPath("warehouse"),
					freight.Origin.Name,
					"warehouse does not exist",
				),
			},
		)
	}

	if err := w.validateFreightArtifactsFn(freight, warehouse); err != nil {
		return nil, err
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

	// Freight is meant to be immutable.
	if changedPath, change, ok := compareFreight(oldFreight, newFreight); !ok {
		return nil, apierrors.NewInvalid(
			freightGroupKind,
			oldFreight.Name,
			field.ErrorList{
				field.Invalid(
					changedPath,
					change,
					"Freight is immutable",
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
			if !oldFreight.IsApprovedFor(approvedStage) {
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
			indexer.StagesByFreightField: freight.Name,
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
	actor := api.FormatEventKubernetesUserActor(req.UserInfo)
	w.recorder.AnnotatedEventf(
		f,
		api.NewFreightApprovedEventAnnotations(actor, f, stageName),
		corev1.EventTypeNormal,
		kargoapi.EventReasonFreightApproved,
		"Freight approved for Stage %q by %q",
		stageName,
		actor,
	)
}

type artifactType string

func (a artifactType) FreightPath() string {
	switch a {
	case artifactTypeGit:
		return "commits"
	case artifactTypeImage:
		return "images"
	case artifactTypeChart:
		return "charts"
	default:
		return ""
	}
}

const (
	artifactTypeGit   artifactType = "git"
	artifactTypeImage artifactType = "image"
	artifactTypeChart artifactType = "chart"
)

type artifactSubscription struct {
	URL  string
	Type artifactType
}

// validateFreightArtifacts checks that the artifacts in the Freight are all
// subscribed to by the Warehouse. It returns an error if:
//
//   - An artifact in the Freight is not subscribed to by the Warehouse.
//   - An artifact for a subscription of the Warehouse is not found in the Freight.
//   - Multiple artifacts in the Freight correspond to the same subscription.
func validateFreightArtifacts(
	freight *kargoapi.Freight,
	warehouse *kargoapi.Warehouse,
) error {
	var subscriptions = make(map[artifactSubscription]bool, len(warehouse.Spec.Subscriptions))
	var counts = make(map[artifactSubscription]int)

	// Collect all the subscriptions from the Warehouse.
	for _, repo := range warehouse.Spec.Subscriptions {
		if repo.Git != nil {
			subscriptions[artifactSubscription{
				URL:  git.NormalizeURL(repo.Git.RepoURL),
				Type: artifactTypeGit,
			}] = false
		}
		if repo.Image != nil {
			subscriptions[artifactSubscription{
				URL:  repo.Image.RepoURL,
				Type: artifactTypeImage,
			}] = false
		}
		if repo.Chart != nil {
			subscriptions[artifactSubscription{
				URL:  path.Join(helm.NormalizeChartRepositoryURL(repo.Chart.RepoURL), repo.Chart.Name),
				Type: artifactTypeChart,
			}] = false
		}
	}

	// Mark the subscription as found for each artifact in the Freight, and count
	// the number of times each subscription is found.
	for _, commit := range freight.Commits {
		sub := artifactSubscription{
			URL:  git.NormalizeURL(commit.RepoURL),
			Type: artifactTypeGit,
		}
		if _, ok := subscriptions[sub]; ok {
			subscriptions[sub] = true
			counts[sub]++
			continue
		}
		return apierrors.NewInvalid(
			freightGroupKind,
			freight.Name,
			field.ErrorList{
				field.Invalid(
					field.NewPath("commits"),
					commit,
					fmt.Sprintf("no subscription found for Git repository in Warehouse %q", warehouse.Name),
				),
			},
		)
	}
	for _, image := range freight.Images {
		sub := artifactSubscription{
			URL:  image.RepoURL,
			Type: artifactTypeImage,
		}
		if _, ok := subscriptions[sub]; ok {
			subscriptions[sub] = true
			counts[sub]++
			continue
		}
		return apierrors.NewInvalid(
			freightGroupKind,
			freight.Name,
			field.ErrorList{
				field.Invalid(
					field.NewPath("images"),
					image,
					fmt.Sprintf("no subscription found for image repository in Warehouse %q", warehouse.Name),
				),
			},
		)
	}
	for _, chart := range freight.Charts {
		sub := artifactSubscription{
			URL:  path.Join(helm.NormalizeChartRepositoryURL(chart.RepoURL), chart.Name),
			Type: artifactTypeChart,
		}
		if _, ok := subscriptions[sub]; ok {
			subscriptions[sub] = true
			counts[sub]++
			continue
		}
		return apierrors.NewInvalid(
			freightGroupKind,
			freight.Name,
			field.ErrorList{
				field.Invalid(
					field.NewPath("charts"),
					chart,
					fmt.Sprintf("no subscription found for Helm chart in Warehouse %q", warehouse.Name),
				),
			},
		)
	}

	// Check that each subscription is found exactly once.
	for sub, found := range subscriptions {
		if !found {
			return apierrors.NewInvalid(
				freightGroupKind,
				freight.Name,
				field.ErrorList{
					field.Invalid(
						field.NewPath(sub.Type.FreightPath()),
						nil,
						fmt.Sprintf(
							"no artifact found for subscription %q of Warehouse %q",
							sub.URL, warehouse.Name,
						),
					),
				},
			)
		}
		if counts[sub] > 1 {
			return apierrors.NewInvalid(
				freightGroupKind,
				freight.Name,
				field.ErrorList{
					field.Invalid(
						field.NewPath(sub.Type.FreightPath()),
						nil,
						fmt.Sprintf(
							"multiple artifacts found for subscription %q of Warehouse %q",
							sub.URL, warehouse.Name,
						),
					),
				},
			)
		}
	}

	return nil
}

// compareFreight compares two Freight objects and returns the first field path
// that differs between them, the new value, and a boolean indicating whether
// the two Freight objects are equal.
func compareFreight(existing, updated *kargoapi.Freight) (*field.Path, any, bool) {
	if !existing.Origin.Equals(&updated.Origin) {
		return field.NewPath("origin"), updated.Origin, false
	}

	if len(existing.Commits) != len(updated.Commits) {
		return field.NewPath("commits"), updated.Commits, false
	}
	for i, commit := range existing.Commits {
		if !commit.DeepEquals(&updated.Commits[i]) {
			return field.NewPath("commits").Index(i), updated.Commits[i], false
		}
	}

	if len(existing.Images) != len(updated.Images) {
		return field.NewPath("images"), updated.Images, false
	}
	for i, image := range existing.Images {
		if !image.DeepEquals(&updated.Images[i]) {
			return field.NewPath("images").Index(i), updated.Images[i], false
		}
	}

	if len(existing.Charts) != len(updated.Charts) {
		return field.NewPath("charts"), updated.Charts, false
	}
	for i, chart := range existing.Charts {
		if !chart.DeepEquals(&updated.Charts[i]) {
			return field.NewPath("charts").Index(i), updated.Charts[i], false
		}
	}

	return nil, nil, true
}
