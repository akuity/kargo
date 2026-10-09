package api

import (
	"fmt"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// TargetSelectorsForStage parses a Stage's target selectors into label
// selectors. It returns nil for a classic Stage -- one with no targets block --
// which governs no Targets. An empty selector within the list parses to one
// that matches everything, so it selects every Target in the Project.
func TargetSelectorsForStage(stage *kargoapi.Stage) ([]labels.Selector, error) {
	if !stage.IsTargetAware() {
		return nil, nil
	}
	selectors := make([]labels.Selector, 0, len(stage.Spec.Targets.Selectors))
	for i := range stage.Spec.Targets.Selectors {
		selector, err := metav1.LabelSelectorAsSelector(
			&stage.Spec.Targets.Selectors[i],
		)
		if err != nil {
			return nil, fmt.Errorf("error parsing target selector %d: %w", i, err)
		}
		selectors = append(selectors, selector)
	}
	return selectors, nil
}

// AnySelectorMatches returns true if any of the provided selectors matches the
// provided labels. A Target is governed by a Stage when it matches any one of
// the Stage's selectors, so this is the test for governance once the selectors
// have been parsed. It returns false when there are no selectors.
func AnySelectorMatches(selectors []labels.Selector, lbls map[string]string) bool {
	set := labels.Set(lbls)
	for _, selector := range selectors {
		if selector.Matches(set) {
			return true
		}
	}
	return false
}

// FilterTargetsForStage returns, from the given Targets, those the Stage
// governs: the ones matching any of its target selectors, each listed once
// and sorted by name so that repeated calls agree on ordering. Callers pass
// the Targets of the Stage's own Project.
//
// A classic Stage -- one with no targets block -- governs no Targets, and
// this returns nil for it. A target-aware Stage yields an empty, non-nil
// slice when nothing matches. An empty selector within the list selects
// every Target in the Project.
func FilterTargetsForStage(
	stage *kargoapi.Stage,
	targets []kargoapi.Target,
) ([]kargoapi.Target, error) {
	selectors, err := TargetSelectorsForStage(stage)
	if err != nil || selectors == nil {
		return nil, err
	}
	// A Target matching more than one selector must still be governed once.
	seen := make(map[string]struct{}, len(targets))
	governed := make([]kargoapi.Target, 0, len(targets))
	for _, target := range targets {
		if _, ok := seen[target.Name]; ok {
			continue
		}
		if !AnySelectorMatches(selectors, target.Labels) {
			continue
		}
		seen[target.Name] = struct{}{}
		governed = append(governed, target)
	}
	slices.SortFunc(governed, func(lhs, rhs kargoapi.Target) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})
	return governed, nil
}
