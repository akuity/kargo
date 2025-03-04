package v1alpha1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/server/user"
)

// VerificationRequest is a request payload with an optional actor field which
// can be used to annotate a Stage using the AnnotationKeyReverify or
// AnnotationKeyAbort annotations.
//
// The ID field is used to specify the VerificationInfo to be re-verified or
// aborted. If the ID is empty, the request is considered invalid. The Actor
// field is optional and can be used to track the user who initiated the
// re-verification or abort request. The ControlPlane field is optional and
// indicates if the request was initiated by a control plane.
//
// +protobuf=false
// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
type VerificationRequest struct {
	// ID is the identifier of the VerificationInfo for which the request is
	// being made.
	ID string `json:"id,omitempty"`
	// Actor is the user who initiated the request.
	Actor string `json:"actor,omitempty"`
	// ControlPlane is a flag to indicate if the request has been initiated by
	// a control plane.
	ControlPlane bool `json:"controlPlane,omitempty"`
}

// Equals returns true if the VerificationRequest is equal to the other
// VerificationRequest, false otherwise. Two VerificationRequests are equal if
// their ID, Actor, and ControlPlane fields are equal.
func (r *VerificationRequest) Equals(other *VerificationRequest) bool {
	if r == nil && other == nil {
		return true
	}
	if r == nil || other == nil {
		return false
	}
	return r.ID == other.ID && r.Actor == other.Actor && r.ControlPlane == other.ControlPlane
}

// HasID returns true if the VerificationRequest has a non-empty ID.
func (r *VerificationRequest) HasID() bool {
	return r != nil && r.ID != ""
}

// ForID returns true if the VerificationRequest has the specified ID.
func (r *VerificationRequest) ForID(id string) bool {
	return r != nil && r.ID != "" && r.ID == id
}

// String returns the JSON string representation of the VerificationRequest,
// or an empty string if the VerificationRequest is nil or has an empty ID.
func (r *VerificationRequest) String() string {
	if r == nil || r.ID == "" {
		return ""
	}
	b, _ := json.Marshal(r)
	if b == nil {
		return ""
	}
	return string(b)
}

// GetStage returns a pointer to the Stage resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func GetStage(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*Stage, error) {
	stage := Stage{}
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

// ListAvailableFreight lists all Freight available to the Stage for any reason.
// This includes:
//
//  1. Any Freight from a Warehouse that the Stage subscribes to directly
//  2. Any Freight that is verified in upstream Stages matching configured AvailabilityStrategy
//     (with any applicable soak time elapsed)
//  3. Any Freight that is approved for the Stage
func (s *Stage) ListAvailableFreight(
	ctx context.Context,
	c client.Client,
) ([]Freight, error) {
	availableFreight := []Freight{}

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
		freightFromWarehouse, err := warehouse.ListFreight(ctx, c, listOpts)
		if err != nil {
			return nil, err
		}
		availableFreight = append(availableFreight, freightFromWarehouse...)
	}

	// Sort and de-dupe the available Freight
	slices.SortFunc(availableFreight, func(lhs, rhs Freight) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})
	availableFreight = slices.CompactFunc(availableFreight, func(lhs, rhs Freight) bool {
		return lhs.Name == rhs.Name
	})

	return availableFreight, nil
}

// IsFreightAvailable answers whether the specified Freight is available to the
// Stage.
func (s *Stage) IsFreightAvailable(freight *Freight) bool {
	if s == nil || freight == nil || s.Namespace != freight.Namespace {
		return false
	}
	if freight.IsApprovedFor(s.Name) {
		return true
	}
	for _, req := range s.Spec.RequestedFreight {
		if freight.Origin.Equals(&req.Origin) {
			if req.Sources.Direct {
				return true
			}
			for _, source := range req.Sources.Stages {
				if freight.IsVerifiedIn(source) {
					return req.Sources.RequiredSoakTime == nil ||
						freight.GetLongestSoak(source) >= req.Sources.RequiredSoakTime.Duration
				}
			}
		}
	}
	return false
}

// RefreshStage forces reconciliation of a Stage by setting an annotation
// on the Stage, causing the controller to reconcile it. Currently, the
// annotation value is the timestamp of the request, but might in the
// future include additional metadata/context necessary for the request.
func RefreshStage(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*Stage, error) {
	stage := &Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespacedName.Namespace,
			Name:      namespacedName.Name,
		},
	}
	if err := patchAnnotation(ctx, c, stage, AnnotationKeyRefresh, time.Now().Format(time.RFC3339)); err != nil {
		return nil, fmt.Errorf("refresh: %w", err)
	}
	return stage, nil
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

	rr := VerificationRequest{
		ID: currentVI.ID,
	}
	// Put actor information to track on the controller side
	if u, ok := user.InfoFromContext(ctx); ok {
		rr.Actor = FormatEventUserActor(u)
	}
	return patchAnnotation(ctx, c, stage, AnnotationKeyReverify, rr.String())
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

	ar := VerificationRequest{
		ID: currentVI.ID,
	}
	// Put actor information to track on the controller side
	if u, ok := user.InfoFromContext(ctx); ok {
		ar.Actor = FormatEventUserActor(u)
	}
	return patchAnnotation(ctx, c, stage, AnnotationKeyAbort, ar.String())
}
