package controller

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// GetShardPredicate constructs a predicate used as an event filter for various
// reconcilers. If a non-empty shard name is passed to this function, it returns
// a predicate that matches ONLY resources labeled for that shard. If an empty
// shard name is passed to this function, it returns a predicate that matches
// ONLY resources that are NOT labeled for ANY shard.
func GetShardPredicate(shard string) (predicate.Predicate, error) {
	if shard == "" {
		pred, err := predicate.LabelSelectorPredicate(
			metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      kargoapi.ShardLabelKey,
						Operator: metav1.LabelSelectorOpDoesNotExist,
					},
				},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("error creating default selector predicate: %w", err)
		}
		return pred, nil
	}
	pred, err := predicate.LabelSelectorPredicate(
		*metav1.SetAsLabelSelector(
			labels.Set(
				map[string]string{
					kargoapi.ShardLabelKey: shard,
				},
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating shard selector predicate: %w", err)
	}
	return pred, nil
}

func GetShardRequirement(shard string) (*labels.Requirement, error) {
	req, err := labels.NewRequirement(kargoapi.ShardLabelKey, selection.Equals, []string{shard})
	if err != nil {
		return nil, fmt.Errorf("error creating shard label selector: %w", err)
	}
	if shard == "" {
		req, err = labels.NewRequirement(kargoapi.ShardLabelKey, selection.DoesNotExist, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating default label selector: %w", err)
		}
	}

	return req, nil
}
