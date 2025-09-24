package controller

import (
	"fmt"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// GetShardRequirement considers the provided shard name and boolean indicating
// whether this controller is the default controller and attempts to distill
// this information down to a labels.Requirement. In cases where it is unable to
// do so, it will return a nil labels.Requirement.
func GetShardRequirement(
	shard string,
	isDefaultController bool,
) (*labels.Requirement, error) {
	if shard != "" {
		if isDefaultController {
			// This controller is responsible for reconciling resources labeled for a
			// specific shard as well as those NOT labeled for any shard. Kubernetes
			// lacks a way to express a requirement that a label either have a certain
			// value or be absent altogether, so for this case, we return no label
			// requirements. Individual reconcilers will have to apply a predicate to
			// filter out resources they're not responsible for.
			return nil, nil
		}
		// This controller is only responsible for reconciling resources labeled for
		// a specific shard.
		req, err := labels.NewRequirement(
			kargoapi.LabelKeyShard,
			selection.Equals,
			[]string{shard},
		)
		if err != nil {
			return nil, fmt.Errorf("error creating shard label selector: %w", err)
		}
		return req, nil
	}
	// This controller is only responsible for reconciling resources not labeled
	// for a specific shard.
	req, err := labels.NewRequirement(
		kargoapi.LabelKeyShard,
		selection.DoesNotExist,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating default label selector: %w", err)
	}
	return req, nil
}
