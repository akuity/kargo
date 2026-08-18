package api

import (
	"context"
	"fmt"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// ListTargetsForStage returns the Targets in the Stage's own Project that the
// Stage governs, i.e. those matching any of its target selectors. Results are
// sorted by name so that repeated calls agree on ordering.
//
// A classic Stage -- one with no targets block -- governs no Targets, and this
// returns an empty slice for it. An empty selector within the list selects
// every Target in the Project.
func ListTargetsForStage(
	ctx context.Context,
	c client.Client,
	stage *kargoapi.Stage,
) ([]kargoapi.Target, error) {
	if !IsTargetAware(stage) {
		return nil, nil
	}

	// A Target matching more than one selector must still be governed once.
	seen := make(map[string]struct{})
	var targets []kargoapi.Target

	for i := range stage.Spec.Targets.Selectors {
		selector, err := metav1.LabelSelectorAsSelector(
			&stage.Spec.Targets.Selectors[i],
		)
		if err != nil {
			return nil, fmt.Errorf("error parsing target selector %d: %w", i, err)
		}

		list := kargoapi.TargetList{}
		if err = c.List(
			ctx,
			&list,
			client.InNamespace(stage.Namespace),
			client.MatchingLabelsSelector{Selector: selector},
		); err != nil {
			return nil, fmt.Errorf(
				"error listing Targets in namespace %q: %w",
				stage.Namespace, err,
			)
		}

		for _, target := range list.Items {
			if _, ok := seen[target.Name]; ok {
				continue
			}
			seen[target.Name] = struct{}{}
			targets = append(targets, target)
		}
	}

	slices.SortFunc(targets, func(lhs, rhs kargoapi.Target) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})
	return targets, nil
}
