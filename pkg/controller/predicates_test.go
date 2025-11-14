package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestResponsibleFor(t *testing.T) {
	testCases := []struct {
		name                string
		shardName           string
		isDefaultController bool
		labels              labels.Set
		responsibleFor      bool
	}{
		{
			name:                "no shard name + default controller + labeled",
			shardName:           "",
			isDefaultController: true,
			labels:              labels.Set{kargoapi.LabelKeyShard: "fake-shard"},
			responsibleFor:      false,
		},
		{
			name:                "no shard name + default controller + unlabeled",
			shardName:           "",
			isDefaultController: true,
			labels:              labels.Set{},
			responsibleFor:      true,
		},
		{
			name:                "no shard name + not default controller + labeled",
			shardName:           "", // Still the de factor default!
			isDefaultController: false,
			labels:              labels.Set{kargoapi.LabelKeyShard: "fake-shard"},
			responsibleFor:      false,
		},
		{
			name:                "no shard name + not default controller + unlabeled",
			shardName:           "", // Still the de factor default!
			isDefaultController: false,
			labels:              labels.Set{},
			responsibleFor:      true,
		},
		{
			name:                "shard name + default controller + wrong label",
			shardName:           "right-shard",
			isDefaultController: true,
			labels:              labels.Set{kargoapi.LabelKeyShard: "wrong-shard"},
			responsibleFor:      false,
		},
		{
			name:                "shard name + default controller + right label",
			shardName:           "right-shard",
			isDefaultController: true,
			labels:              labels.Set{kargoapi.LabelKeyShard: "right-shard"},
			responsibleFor:      true,
		},
		{
			name:                "shard name + default controller + no label",
			shardName:           "right-shard",
			isDefaultController: true,
			labels:              labels.Set{},
			responsibleFor:      true,
		},
		{
			name:                "shard name + not default controller + wrong label",
			shardName:           "right-shard",
			isDefaultController: false,
			labels:              labels.Set{kargoapi.LabelKeyShard: "wrong-shard"},
			responsibleFor:      false,
		},
		{
			name:                "shard name + not default controller + right label",
			shardName:           "right-shard",
			isDefaultController: false,
			labels:              labels.Set{kargoapi.LabelKeyShard: "right-shard"},
			responsibleFor:      true,
		},
		{
			name:                "shard name + not default controller + no label",
			shardName:           "right-shard",
			isDefaultController: false,
			labels:              labels.Set{},
			responsibleFor:      false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.responsibleFor,
				(&ResponsibleFor[kargoapi.Stage]{
					IsDefaultController: testCase.isDefaultController,
					ShardName:           testCase.shardName,
				}).IsResponsible(&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{Labels: testCase.labels},
				}),
			)
		})
	}
}
