package api

import (
	"context"
	"fmt"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// ResolveStageTargets returns the Targets governed by a Stage. A nil result
// means the Stage is in classic mode; an empty, non-nil result means the Stage
// explicitly governs Targets but none currently match its selectors.
func ResolveStageTargets(
	ctx context.Context,
	reader client.Reader,
	stage *kargoapi.Stage,
) ([]kargoapi.Target, error) {
	if stage.Spec.TargetSelectors == nil {
		return nil, nil
	}

	selectors := make([]labels.Selector, 0, len(stage.Spec.TargetSelectors))
	for _, labelSelector := range stage.Spec.TargetSelectors {
		selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
		if err != nil {
			return nil, fmt.Errorf("convert target selector: %w", err)
		}
		selectors = append(selectors, selector)
	}

	targets := &kargoapi.TargetList{}
	if err := reader.List(ctx, targets, client.InNamespace(stage.Namespace)); err != nil {
		return nil, fmt.Errorf("list targets: %w", err)
	}

	result := make([]kargoapi.Target, 0, len(targets.Items))
	for _, target := range targets.Items {
		for _, selector := range selectors {
			if selector.Matches(labels.Set(target.Labels)) {
				result = append(result, target)
				break
			}
		}
	}
	slices.SortFunc(result, func(a, b kargoapi.Target) int {
		return strings.Compare(a.Name, b.Name)
	})
	return result, nil
}
