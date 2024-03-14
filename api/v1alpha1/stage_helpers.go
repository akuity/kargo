package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

// ClearStageRefresh is called by the Stage controller to clear the refresh
// annotation on the Stage (if present). A client (e.g. UI) who requested a
// Stage refresh, can wait until the annotation is cleared, to understand that
// the controller successfully reconciled the Stage after the refresh request.
func ClearStageRefresh(
	ctx context.Context,
	c client.Client,
	stage *Stage,
) error {
	if stage.Annotations == nil {
		return nil
	}
	if _, ok := stage.Annotations[AnnotationKeyRefresh]; !ok {
		return nil
	}
	newStage := Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stage.Name,
			Namespace: stage.Namespace,
		},
	}
	return clearObjectAnnotation(ctx, c, &newStage, AnnotationKeyRefresh)
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

	return patchAnnotation(ctx, c, stage, AnnotationKeyReverify, curFreight.VerificationInfo.ID)
}

// ClearStageReverify is called by the Stage controller to clear the
// AnnotationKeyReverify annotation on the Stage (if present). A client (e.g.
// UI) who requested a reconfirmation of the Stage verification, can wait
// until the annotation is cleared, to understand that the controller
// acknowledged the reverification request.
func ClearStageReverify(
	ctx context.Context,
	c client.Client,
	stage *Stage,
) error {
	if stage.Annotations == nil {
		return nil
	}

	if _, ok := stage.Annotations[AnnotationKeyReverify]; !ok {
		return nil
	}

	newStage := Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stage.Name,
			Namespace: stage.Namespace,
		},
	}
	return clearObjectAnnotation(ctx, c, &newStage, AnnotationKeyReverify)
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

// ClearStageAbort is called by the Stage controller to clear the
// AnnotationKeyAbort annotation on the Stage (if present). A client (e.g.
// UI) who requested an abort of the Stage verification, can wait
// until the annotation is cleared, to understand that the controller
// acknowledged the abort request.
func ClearStageAbort(
	ctx context.Context,
	c client.Client,
	stage *Stage,
) error {
	if stage.Annotations == nil {
		return nil
	}

	if _, ok := stage.Annotations[AnnotationKeyAbort]; !ok {
		return nil
	}

	newStage := Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stage.Name,
			Namespace: stage.Namespace,
		},
	}
	return clearObjectAnnotation(ctx, c, &newStage, AnnotationKeyAbort)
}
