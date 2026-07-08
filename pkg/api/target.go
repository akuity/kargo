package api

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// GetTargetsForStage returns the Target(s) that Freight promoted to the given
// Stage is destined for.
//
// When the Stage defines a TargetSelector, the Target resources in the Stage's
// namespace whose labels satisfy the selector are returned. Otherwise a single
// ephemeral Target -- synthesized from the Stage and NOT persisted -- is
// returned, representing the Stage's default destination.
func GetTargetsForStage(
	ctx context.Context,
	c client.Client,
	stage *kargoapi.Stage,
) ([]kargoapi.Target, error) {
	if stage.Spec.TargetSelector == nil {
		return []kargoapi.Target{defaultTargetForStage(stage)}, nil
	}

	selector, err := metav1.LabelSelectorAsSelector(stage.Spec.TargetSelector)
	if err != nil {
		return nil, fmt.Errorf(
			"error parsing target selector for Stage %q in namespace %q: %w",
			stage.Name, stage.Namespace, err,
		)
	}

	list := &kargoapi.TargetList{}
	if err = c.List(
		ctx,
		list,
		client.InNamespace(stage.Namespace),
		client.MatchingLabelsSelector{Selector: selector},
	); err != nil {
		return nil, fmt.Errorf(
			"error listing Targets for Stage %q in namespace %q: %w",
			stage.Name, stage.Namespace, err,
		)
	}
	return list.Items, nil
}

// defaultTargetForStage synthesizes the ephemeral, non-persisted Target used
// when a Stage does not define a TargetSelector. It shares the Stage's name and
// namespace and carries the Stage label (plus the Stage's shard label, when
// present) so it is indistinguishable from a Target a user might define for the
// same destination.
func defaultTargetForStage(stage *kargoapi.Stage) kargoapi.Target {
	labels := map[string]string{
		kargoapi.LabelKeyStage: stage.Name,
	}
	if shard := stage.Labels[kargoapi.LabelKeyShard]; shard != "" {
		labels[kargoapi.LabelKeyShard] = shard
	}
	return kargoapi.Target{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stage.Name,
			Namespace: stage.Namespace,
			Labels:    labels,
		},
	}
}
