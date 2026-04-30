package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/labels"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestGetShardRequirement(t *testing.T) {
	testCases := []struct {
		name                string
		shardName           string
		isDefaultController bool
		assertions          func(*testing.T, *labels.Requirement, error)
	}{
		{
			name:                "no shard name + default controller",
			shardName:           "",
			isDefaultController: true,
			assertions: func(t *testing.T, req *labels.Requirement, err error) {
				require.NoError(t, err)
				require.NotNil(t, req)
				require.True(t, req.Matches(labels.Set{}))
				require.False(
					t,
					req.Matches(labels.Set{kargoapi.LabelKeyShard: "fake-shard"}),
				)
			},
		},
		{
			name:                "no shard name + not default controller",
			shardName:           "",
			isDefaultController: false,
			assertions: func(t *testing.T, req *labels.Requirement, err error) {
				require.NoError(t, err)
				// Absence of a shard name makes this controller the de facto default.
				// It doesn't matter that isDefaultController is false. If Kargo is
				// installed via its Helm chart, this combination of settings should
				// actually never occur.
				require.NotNil(t, req)
				require.True(t, req.Matches(labels.Set{}))
				require.False(
					t,
					req.Matches(labels.Set{kargoapi.LabelKeyShard: "fake-shard"}),
				)
			},
		},
		{
			name:                "shard name + default controller",
			shardName:           "fake-shard",
			isDefaultController: true,
			assertions: func(t *testing.T, req *labels.Requirement, err error) {
				require.NoError(t, err)
				// These conditions cannot be distilled down to a single
				// labels.Requirement, so we expect none is returned. The caller will
				// determine how to proceed under these circumstances.
				require.Nil(t, req)
			},
		},
		{
			name:                "shard name + not default controller",
			shardName:           "fake-shard",
			isDefaultController: false,
			assertions: func(t *testing.T, req *labels.Requirement, err error) {
				require.NoError(t, err)
				require.NotNil(t, req)
				require.False(t, req.Matches(labels.Set{}))
				require.True(
					t,
					req.Matches(labels.Set{kargoapi.LabelKeyShard: "fake-shard"}),
				)
				require.False(
					t,
					req.Matches(labels.Set{
						kargoapi.LabelKeyShard: "different-fake-shard",
					}),
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			req, err := GetShardRequirement(
				testCase.shardName,
				testCase.isDefaultController,
			)
			testCase.assertions(t, req, err)
		})
	}
}
