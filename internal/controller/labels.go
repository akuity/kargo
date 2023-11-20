package controller

import (
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const ShardLabelKey = "kargo.akuity.io/shard"

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
						Key:      ShardLabelKey,
						Operator: metav1.LabelSelectorOpDoesNotExist,
					},
				},
			},
		)
		return pred, errors.Wrap(err, "error creating default selector predicate")
	}
	pred, err := predicate.LabelSelectorPredicate(
		*metav1.SetAsLabelSelector(
			labels.Set(
				map[string]string{
					ShardLabelKey: shard,
				},
			),
		),
	)
	return pred, errors.Wrap(err, "error creating shard selector predicate")
}
