package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/server/user"
)

// AbortAction is an action to take on a Promotion to abort it.
type AbortAction string

const (
	// AbortActionTerminate is an action to terminate the Promotion.
	// I.e. the Promotion will be marked as failed and the controller
	// will stop processing it.
	AbortActionTerminate AbortAction = "terminate"
)

// AbortPromotionRequest is a request payload with an optional actor field which
// can be used to annotate a Promotion using the AnnotationKeyAbort annotation.
//
// +protobuf=false
// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
type AbortPromotionRequest struct {
	// Action is the action to take on the Promotion to abort it.
	Action AbortAction `json:"action,omitempty" protobuf:"bytes,1,opt,name=action"`
	// Actor is the user who initiated the request.
	Actor string `json:"actor,omitempty" protobuf:"bytes,2,opt,name=actor"`
	// ControlPlane is a flag to indicate if the request has been initiated by
	// a control plane.
	ControlPlane bool `json:"controlPlane,omitempty" protobuf:"varint,3,opt,name=controlPlane"`
}

// Equals returns true if the AbortPromotionRequest is equal to the other
// AbortPromotionRequest, false otherwise. Two VerificationRequests are equal
// if their Action, Actor, and ControlPlane fields are equal.
func (r *AbortPromotionRequest) Equals(other *AbortPromotionRequest) bool {
	if r == nil && other == nil {
		return true
	}
	if r == nil || other == nil {
		return false
	}
	return r.Action == other.Action && r.Actor == other.Actor && r.ControlPlane == other.ControlPlane
}

// String returns the JSON string representation of the AbortPromotionRequest,
// or an empty string if the AbortPromotionRequest is nil or has an empty Action.
func (r *AbortPromotionRequest) String() string {
	if r == nil || r.Action == "" {
		return ""
	}
	b, _ := json.Marshal(r)
	if b == nil {
		return ""
	}
	return string(b)
}

// GetPromotion returns a pointer to the Promotion resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func GetPromotion(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*Promotion, error) {
	promo := Promotion{}
	if err := c.Get(ctx, namespacedName, &promo); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"error getting Promotion %q in namespace %q: %w",
			namespacedName.Name,
			namespacedName.Namespace,
			err,
		)
	}
	return &promo, nil
}

// RefreshPromotion forces reconciliation of a Promotion by setting an annotation
// on the Promotion, causing the controller to reconcile it. Currently, the
// annotation value is the timestamp of the request, but might in the
// future include additional metadata/context necessary for the request.
func RefreshPromotion(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*Promotion, error) {
	promo := &Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespacedName.Namespace,
			Name:      namespacedName.Name,
		},
	}
	if err := patchAnnotation(ctx, c, promo, AnnotationKeyRefresh, time.Now().Format(time.RFC3339)); err != nil {
		return nil, fmt.Errorf("refresh: %w", err)
	}
	return promo, nil
}

// AbortPromotion forces aborting the Promotion by setting an annotation on the
// object, causing the controller to abort the Promotion. The annotation value
// is the action to take on the Promotion to abort it.
func AbortPromotion(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
	action AbortAction,
) error {
	promotion, err := GetPromotion(ctx, c, namespacedName)
	if err != nil || promotion == nil {
		if promotion == nil {
			err = fmt.Errorf(
				"Promotion %q in namespace %q not found",
				namespacedName.Name,
				namespacedName.Namespace,
			)
		}
		return err
	}

	if promotion.Status.Phase.IsTerminal() {
		// The Promotion is already in a terminal phase, so we can skip the
		// abort request.
		return nil
	}

	ar := AbortPromotionRequest{
		Action: action,
	}
	// Put actor information to track on the controller side
	if u, ok := user.InfoFromContext(ctx); ok {
		ar.Actor = FormatEventUserActor(u)
	}
	return patchAnnotation(ctx, c, promotion, AnnotationKeyAbort, ar.String())
}

// ComparePromotionByPhaseAndCreationTime compares two Promotions by their
// phase and creation timestamp. It returns a negative value if Promotion `a`
// should come before Promotion `b`, a positive value if Promotion `a` should
// come after Promotion `b`, or zero if they are considered equal for sorting
// purposes. It can be used in conjunction with slices.SortFunc to sort a list
// of Promotions.
//
// The order of Promotions is as follows:
//  1. Running Promotions
//  2. Non-terminal Promotions (ordered by ULID in ascending order)
//  3. Terminal Promotions (ordered by ULID in descending order)
func ComparePromotionByPhaseAndCreationTime(a, b Promotion) int {
	// Compare the phases of the Promotions first.
	if phaseCompare := ComparePromotionPhase(a.Status.Phase, b.Status.Phase); phaseCompare != 0 {
		return phaseCompare
	}

	switch {
	case !a.Status.Phase.IsTerminal():
		// Non-terminal Promotions are ordered in ascending order based on the
		// ULID in the Promotion name. This ensures that the Promotion which
		// was (or will be) enqueued first is at the top.
		return strings.Compare(a.Name, b.Name)
	default:
		// Terminal Promotions are ordered in descending order based on the
		// ULID in the Promotion name. This ensures that the most recent
		// Promotion is at the top, limiting the number of Promotions which
		// have to be further inspected to collect the "new" Promotions.
		return strings.Compare(b.Name, a.Name)
	}
}

// ComparePromotionPhase compares two Promotion phases. It returns a negative
// value if phase `a` should come before phase `b`, a positive value if phase
// `a` should come after phase `b`, or zero if they are considered equal for
// sorting purposes. It can be used in combination with slices.SortFunc to sort
// a list of Promotion phases.
//
// The order of Promotion phases is as follows:
//  1. Running
//  2. Non-terminal phases
//  3. Terminal phases
func ComparePromotionPhase(a, b PromotionPhase) int {
	aRunning, bRunning := a == PromotionPhaseRunning, b == PromotionPhaseRunning
	aTerminal, bTerminal := a.IsTerminal(), b.IsTerminal()

	// NB: The order of the cases here is important, as "Running" is a special
	// case that should always come before any other phase.
	switch {
	case aRunning && !bRunning:
		return -1
	case !aRunning && bRunning:
		return 1
	case !aTerminal && bTerminal:
		return -1
	case aTerminal && !bTerminal:
		return 1
	default:
		return 0
	}
}
