package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/user"
)

// GetStage returns a pointer to the Stage resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func GetStage(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*kargoapi.Stage, error) {
	stage := kargoapi.Stage{}
	if err := c.Get(ctx, namespacedName, &stage); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"error getting Stage %q in namespace %q: %w",
			namespacedName.Name,
			namespacedName.Namespace,
			err,
		)
	}
	return &stage, nil
}

// ListStagesOptions defines the options for listing Stages.
type ListStagesOptions struct {
	// Warehouses is an optional list of Warehouse names to filter the Stages by.
	Warehouses []string
}

// ListStagesByWarehouses lists Stages in the given Project, optionally
// filtered by the provided options.
func ListStagesByWarehouses(
	ctx context.Context,
	c client.Client,
	project string,
	opts *ListStagesOptions,
) ([]kargoapi.Stage, error) {
	if opts == nil {
		opts = &ListStagesOptions{}
	}
	var list kargoapi.StageList
	if err := c.List(ctx, &list, client.InNamespace(project)); err != nil {
		return nil, err
	}
	if len(opts.Warehouses) == 0 {
		return list.Items, nil
	}
	var stages []kargoapi.Stage
	for _, stage := range list.Items {
		if StageMatchesAnyWarehouse(&stage, opts.Warehouses) {
			stages = append(stages, stage)
		}
	}
	return stages, nil
}

// StageMatchesAnyWarehouse returns true if the Stage requests Freight that
// originated from at least one of the specified warehouses, either directly
// or through upstream stages.
func StageMatchesAnyWarehouse(stage *kargoapi.Stage, warehouses []string) bool {
	for _, req := range stage.Spec.RequestedFreight {
		if req.Origin.Kind == kargoapi.FreightOriginKindWarehouse &&
			slices.Contains(warehouses, req.Origin.Name) {
			return true
		}
	}
	return false
}

// ListFreightAvailableToStage lists all Freight available to the Stage for any
// reason. This includes:
//
//  1. Any Freight from a Warehouse that the Stage subscribes to directly
//  2. Any Freight that is verified in upstream Stages matching configured
//     AvailabilityStrategy (with any applicable soak time elapsed)
//  3. Any Freight that is approved for the Stage
func ListFreightAvailableToStage(
	ctx context.Context,
	c client.Client,
	s *kargoapi.Stage,
) ([]kargoapi.Freight, error) {
	var availableFreight []kargoapi.Freight

	for _, req := range s.Spec.RequestedFreight {
		// Get the Warehouse of origin
		warehouse, err := GetWarehouse(
			ctx,
			c,
			types.NamespacedName{
				Namespace: s.Namespace,
				Name:      req.Origin.Name,
			},
		)
		if err != nil {
			return nil, err
		}
		if warehouse == nil {
			// nolint:staticcheck
			return nil, fmt.Errorf(
				"Warehouse %q not found in namespace %q",
				req.Origin.Name,
				s.Namespace,
			)
		}
		// Get applicable Freight from the Warehouse
		var listOpts *ListWarehouseFreightOptions
		if !req.Sources.Direct {
			listOpts = &ListWarehouseFreightOptions{
				ApprovedFor:          s.Name,
				VerifiedIn:           req.Sources.Stages,
				AvailabilityStrategy: req.Sources.AvailabilityStrategy,
				RequiredSoakTime:     req.Sources.RequiredSoakTime,
			}
			if req.Sources.AutoPromotionOptions != nil &&
				req.Sources.AutoPromotionOptions.SelectionPolicy == kargoapi.AutoPromotionSelectionPolicyMatchUpstream {
				// Validation should have ensured there is exactly one upstream Stage
				// if this selection policy is set.
				listOpts.CurrentlyIn = req.Sources.Stages[0]
			}
		}
		freightFromWarehouse, err := ListFreightFromWarehouse(
			ctx, c, warehouse, listOpts,
		)
		if err != nil {
			return nil, err
		}
		availableFreight = append(availableFreight, freightFromWarehouse...)
	}

	// Sort and de-dupe the available Freight
	slices.SortFunc(availableFreight, func(lhs, rhs kargoapi.Freight) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})
	availableFreight = slices.CompactFunc(availableFreight, func(lhs, rhs kargoapi.Freight) bool {
		return lhs.Name == rhs.Name
	})

	return availableFreight, nil
}

// RefreshStage forces reconciliation of a Stage by setting an annotation
// on the Stage, causing the controller to reconcile it. Currently, the
// annotation value is the timestamp of the request, but might in the
// future include additional metadata/context necessary for the request.
func RefreshStage(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*kargoapi.Stage, error) {
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespacedName.Namespace,
			Name:      namespacedName.Name,
		},
	}
	if err := patchAnnotation(ctx, c, stage, kargoapi.AnnotationKeyRefresh, time.Now().Format(time.RFC3339)); err != nil {
		return nil, fmt.Errorf("refresh: %w", err)
	}
	return stage, nil
}

// AnnotateStageWithArgoCDContext annotates a Stage with the ArgoCD context
// necessary for the frontend to display ArgoCD information for the Stage.
//
// The annotation value is a JSON-encoded list of ArgoCD apps that are
// associated with the Stage, constructed from the HealthCheckSteps from
// the latest Promotion.
//
// If no ArgoCD apps are found, the annotation is removed.
func AnnotateStageWithArgoCDContext(
	ctx context.Context,
	c client.Client,
	healthChecks []kargoapi.HealthCheckStep,
	stage *kargoapi.Stage,
) error {
	var argoCDApps []map[string]any
	for _, healthCheck := range healthChecks {
		healthCheckConfig := healthCheck.GetConfig()

		appsList, ok := healthCheckConfig["apps"].([]any)
		if !ok {
			continue
		}

		for _, rawApp := range appsList {
			appConfig, ok := rawApp.(map[string]any)
			if !ok {
				continue
			}
			argoCDApps = append(argoCDApps, map[string]any{
				"name":      appConfig["name"],
				"namespace": appConfig["namespace"],
			})
		}
	}

	// If we did not find any ArgoCD apps, we should remove the annotation.
	if len(argoCDApps) == 0 {
		return deleteAnnotation(ctx, c, stage, kargoapi.AnnotationKeyArgoCDContext)
	}

	// Marshal the ArgoCD context to JSON and set the annotation on the Stage.
	argoCDAppsJSON, err := json.Marshal(argoCDApps)
	if err != nil {
		return fmt.Errorf("failed to marshal ArgoCD context: %w", err)
	}
	return patchAnnotation(ctx, c, stage, kargoapi.AnnotationKeyArgoCDContext, string(argoCDAppsJSON))
}

// ReverifyStageFreight forces reconfirmation of the verification of the
// Freight associated with a Stage by setting an AnnotationKeyReverify
// annotation on the Stage, causing the controller to rerun the verification.
// The annotation value is the identifier of the existing VerificationInfo for
// the Stage.
func ReverifyStageFreight(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) error {
	stage, err := GetStage(ctx, c, namespacedName)
	if err != nil || stage == nil {
		if stage == nil {
			// nolint:staticcheck
			err = fmt.Errorf("Stage %q in namespace %q not found", namespacedName.Name, namespacedName.Namespace)
		}
		return err
	}

	currentFC := stage.Status.FreightHistory.Current()
	if currentFC == nil || len(currentFC.Freight) == 0 {
		return errors.New("stage has no current freight")
	}

	currentVI := currentFC.VerificationHistory.Current()
	if currentVI == nil {
		return errors.New("stage has no current verification info")
	}

	if currentVI.ID == "" {
		return fmt.Errorf("current stage verification info has no ID")
	}

	rr := kargoapi.VerificationRequest{
		ID: currentVI.ID,
	}
	// Put actor information to track on the controller side
	if u, ok := user.InfoFromContext(ctx); ok {
		rr.Actor = FormatEventUserActor(u)
	}
	return patchAnnotation(ctx, c, stage, kargoapi.AnnotationKeyReverify, rr.String())
}

// AbortStageFreightVerification forces aborting the verification of the
// Freight associated with a Stage by setting an AnnotationKeyAbort
// annotation on the Stage, causing the controller to abort the verification.
// The annotation value is the identifier of the existing VerificationInfo for
// the Stage.
func AbortStageFreightVerification(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) error {
	stage, err := GetStage(ctx, c, namespacedName)
	if err != nil || stage == nil {
		if stage == nil {
			// nolint:staticcheck
			err = fmt.Errorf("Stage %q in namespace %q not found", namespacedName.Name, namespacedName.Namespace)
		}
		return err
	}

	currentFC := stage.Status.FreightHistory.Current()
	if currentFC == nil || len(currentFC.Freight) == 0 {
		return errors.New("stage has no current freight")
	}

	currentVI := currentFC.VerificationHistory.Current()
	if currentVI == nil {
		return errors.New("stage has no current verification info")
	}

	if currentVI.Phase.IsTerminal() {
		// The verification is already in a terminal phase, so we can skip the
		// abort request.
		return nil
	}
	if currentVI.ID == "" {
		return fmt.Errorf("current stage verification info has no ID")
	}

	ar := kargoapi.VerificationRequest{
		ID: currentVI.ID,
	}
	// Put actor information to track on the controller side
	if u, ok := user.InfoFromContext(ctx); ok {
		ar.Actor = FormatEventUserActor(u)
	}
	return patchAnnotation(ctx, c, stage, kargoapi.AnnotationKeyAbort, ar.String())
}

// StripStageForSummary mutates the Stage in place, clearing the heavy
// payload fields that list and graph views do not need. The surviving
// shape still preserves has-verification and promotion-step-count
// information (via stage.Spec.Verification != nil and
// len(stage.Spec.PromotionTemplate.Spec.Steps)), so callers do not have
// to refetch via GetStage for those bits.
//
// Stripped fields:
//   - status.freightHistory truncated to the current element (index 0)
//   - spec.promotionTemplate.spec.steps[*].config cleared (kind/as/name kept)
//   - status.health.output cleared (use ListStageHealthOutputs for lazy fetch)
func StripStageForSummary(stage *kargoapi.Stage) {
	if stage == nil {
		return
	}
	if len(stage.Status.FreightHistory) > 1 {
		stage.Status.FreightHistory = stage.Status.FreightHistory[:1]
	}
	if stage.Spec.PromotionTemplate != nil {
		for i := range stage.Spec.PromotionTemplate.Spec.Steps {
			stage.Spec.PromotionTemplate.Spec.Steps[i].Config = nil
		}
	}
	if stage.Status.Health != nil {
		stage.Status.Health.Output = nil
	}
}

// listStageHealthOutputsConcurrency caps the number of concurrent Gets
// issued by ListStageHealthOutputs. Sized to keep the apiserver/SAR fan-out
// bounded for typical viewport-sized batches without serializing them.
const listStageHealthOutputsConcurrency = 16

// ListStageHealthOutputs returns the raw health output blob for each Stage
// in the given project whose name appears in stageNames. Empty and duplicate
// entries in stageNames are ignored. Stages that do not exist in the project
// or have no recorded health output are omitted from the returned map.
//
// Intended for clients that list Stages with the summary projection (see
// StripStageForSummary) and need to lazily resolve per-Stage health only for
// the subset currently in viewport. The implementation reads each requested
// Stage individually so server-side work scales with the request, not the
// project size.
func ListStageHealthOutputs(
	ctx context.Context,
	c client.Client,
	project string,
	stageNames []string,
) (map[string]string, error) {
	seen := make(map[string]struct{}, len(stageNames))
	wanted := make([]string, 0, len(stageNames))
	for _, n := range stageNames {
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		wanted = append(wanted, n)
	}
	if len(wanted) == 0 {
		return map[string]string{}, nil
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(listStageHealthOutputsConcurrency)

	var (
		mu      sync.Mutex
		outputs = make(map[string]string, len(wanted))
	)
	for _, name := range wanted {
		g.Go(func() error {
			var stage kargoapi.Stage
			err := c.Get(
				ctx,
				client.ObjectKey{Namespace: project, Name: name},
				&stage,
			)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}
				return fmt.Errorf("get stage %q: %w", name, err)
			}
			if stage.Status.Health == nil || stage.Status.Health.Output == nil {
				return nil
			}
			raw := string(stage.Status.Health.Output.Raw)
			mu.Lock()
			outputs[name] = raw
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return outputs, nil
}
