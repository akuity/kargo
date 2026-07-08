package stages

import (
	"context"
	"fmt"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

// syncTarget ensures the promotion Target(s) for the given Stage are in the
// desired state, so that a Target is always available by the time the Stage
// runs a Promotion.
//
// When the Stage has no TargetSelector, Kargo manages a single Target for it:
// one sharing the Stage's name and namespace, owned by the Stage (via a
// controller owner reference) so it is garbage-collected with the Stage. Its
// controller-managed metadata is repaired on drift, but its user-editable
// spec.params is never overwritten.
//
// When the Stage has a TargetSelector, it brings its own Target(s) by label, so
// any Target this Stage previously auto-created is removed.
func (r *RegularStageReconciler) syncTarget(
	ctx context.Context,
	stage *kargoapi.Stage,
) error {
	logger := logging.LoggerFromContext(ctx)

	existing := &kargoapi.Target{}
	switch err := r.client.Get(
		ctx,
		client.ObjectKeyFromObject(stage),
		existing,
	); {
	case err == nil:
	case apierrors.IsNotFound(err):
		existing = nil
	default:
		return fmt.Errorf("error getting Target: %w", err)
	}

	// When the Stage selects its Target(s) by label, it manages them itself.
	// Remove any Target we previously auto-created for it.
	if stage.Spec.TargetSelector != nil {
		if existing != nil && isControlledBy(existing, stage) {
			if err := r.client.Delete(ctx, existing); err != nil &&
				!apierrors.IsNotFound(err) {
				return fmt.Errorf("error deleting auto-created Target: %w", err)
			}
			logger.Debug("deleted auto-created Target superseded by targetSelector")
		}
		return nil
	}

	ownerRef := metav1.NewControllerRef(
		stage,
		kargoapi.GroupVersion.WithKind("Stage"),
	)

	// No selector: ensure the auto-created Target exists.
	if existing == nil {
		target := &kargoapi.Target{
			ObjectMeta: metav1.ObjectMeta{
				Name:            stage.Name,
				Namespace:       stage.Namespace,
				Labels:          targetLabels(stage),
				OwnerReferences: []metav1.OwnerReference{*ownerRef},
			},
		}
		if err := r.client.Create(ctx, target); err != nil {
			if apierrors.IsAlreadyExists(err) {
				// A concurrent reconciliation already created it; that is an
				// acceptable outcome.
				return nil
			}
			return fmt.Errorf("error creating Target: %w", err)
		}
		logger.Debug("created Target for Stage")
		return nil
	}

	// The Target already exists. Repair its controller-managed metadata if it
	// has drifted, but never touch the user-editable spec.params.
	patched := existing.DeepCopy()
	if !isControlledBy(patched, stage) {
		patched.OwnerReferences = append(patched.OwnerReferences, *ownerRef)
	}
	if patched.Labels[kargoapi.LabelKeyStage] != stage.Name {
		if patched.Labels == nil {
			patched.Labels = map[string]string{}
		}
		patched.Labels[kargoapi.LabelKeyStage] = stage.Name
	}
	if reflect.DeepEqual(existing.OwnerReferences, patched.OwnerReferences) &&
		reflect.DeepEqual(existing.Labels, patched.Labels) {
		return nil
	}
	if err := r.client.Update(ctx, patched); err != nil {
		return fmt.Errorf("error updating Target: %w", err)
	}
	logger.Debug("repaired managed metadata on existing Target")
	return nil
}

// targetLabels returns the controller-managed labels for a Stage's
// auto-created Target. The Stage's shard label, when present, is mirrored so
// the Target is associated with the same shard.
func targetLabels(stage *kargoapi.Stage) map[string]string {
	labels := map[string]string{
		kargoapi.LabelKeyStage: stage.Name,
	}
	if shard := stage.Labels[kargoapi.LabelKeyShard]; shard != "" {
		labels[kargoapi.LabelKeyShard] = shard
	}
	return labels
}

// isControlledBy reports whether the object's controller owner reference points
// at the given owner.
func isControlledBy(obj, owner metav1.Object) bool {
	ref := metav1.GetControllerOf(obj)
	return ref != nil && ref.UID == owner.GetUID()
}
