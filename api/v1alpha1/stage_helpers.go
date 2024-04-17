package v1alpha1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/api/user"
)

// ReverificationRequest is a request payload with an optional actor field which
// can be used to annotate a Stage using the AnnotationKeyReverify annotation.
// The actor field is used to track the user who initiated the re-verification.
//
// +protobuf=false
// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
type ReverificationRequest struct {
	// ID is the identifier of the VerificationInfo to be re-verified.
	ID string `json:"id,omitempty"`
	// Actor is the user who initiated the re-verification.
	Actor string `json:"actor,omitempty"`
	// ControlPlane is a flag to indicate if the re-verification has been
	// initiated by a control plane.
	ControlPlane bool `json:"controlPlane,omitempty"`
}

// ForID returns true if the ReverificationRequest has the specified ID.
func (r *ReverificationRequest) ForID(id string) bool {
	return r != nil && r.ID == id
}

// String returns the JSON string representation of the ReverificationRequest,
// or an empty string if the ReverificationRequest is nil or empty.
func (r *ReverificationRequest) String() string {
	if r == nil {
		return ""
	}
	b, _ := json.Marshal(r)
	if b == nil || bytes.Equal(b, []byte("{}")) {
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

	curFreight := stage.Status.CurrentFreight
	if curFreight == nil {
		return errors.New("stage has no current freight")
	}
	if curFreight.VerificationInfo == nil {
		return errors.New("stage has no existing verification info")
	}
	if curFreight.VerificationInfo.ID == "" {
		return fmt.Errorf("stage verification info has no ID")
	}

	rr := ReverificationRequest{
		ID: curFreight.VerificationInfo.ID,
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

	curFreight := stage.Status.CurrentFreight
	if curFreight == nil {
		return errors.New("stage has no current freight")
	}
	if curFreight.VerificationInfo == nil {
		return errors.New("stage has no existing verification info")
	}
	if stage.Status.CurrentFreight.VerificationInfo.Phase.IsTerminal() {
		// The verification is already in a terminal phase, so we can skip the
		// abort request.
		return nil
	}
	if curFreight.VerificationInfo.ID == "" {
		return fmt.Errorf("stage verification info has no ID")
	}

	return patchAnnotation(ctx, c, stage, AnnotationKeyAbort, curFreight.VerificationInfo.ID)
}
