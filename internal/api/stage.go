package api

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/server/user"
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
			}
			if requiredSoak := req.Sources.RequiredSoakTime; requiredSoak != nil {
				listOpts.VerifiedBefore = &metav1.Time{Time: time.Now().Add(-requiredSoak.Duration)}
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

func InjectArgoCDContextToStage(
	ctx context.Context,
	c client.Client,
	healthChecks []HealthCheckStep,
	stage *Stage,
) error {
	rawConfigs := []map[string]any{}
	for _, healthCheck := range healthChecks {
		healthCheckConfig := healthCheck.GetConfig()

		apps, validType := healthCheckConfig["apps"].([]interface{})

		if validType {
			for _, untypedApp := range apps {
				app, validTyped := untypedApp.(map[string]interface{})

				if validTyped {
					rawConfigs = append(rawConfigs, map[string]any{
						"name":      app["name"],
						"namespace": app["namespace"],
					})
				}
			}
		}
	}

	configsValue, err := json.Marshal(rawConfigs)

	if err != nil {
		return err
	}

	return patchAnnotation(ctx, c, stage, AnnotationKeyArgoCDContext, string(configsValue))
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
