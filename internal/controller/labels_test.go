package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

func TestGetShardPredicate(t *testing.T) {
	const testShardName = "test-shard"
	unlabeledEvent := event.CreateEvent{
		Object: &kargoapi.Stage{},
	}
	labeledEvent := event.CreateEvent{
		Object: &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					kargoapi.ShardLabelKey: testShardName,
				},
			},
		},
	}
	testCases := []struct {
		name       string
		shardName  string
		assertions func(*testing.T, predicate.Predicate, error)
	}{
		{
			name:      "shard name is the empty string",
			shardName: "",
			assertions: func(t *testing.T, pred predicate.Predicate, err error) {
				require.NoError(t, err)
				require.NotNil(t, pred)
				require.True(t, pred.Create(unlabeledEvent))
				require.False(t, pred.Create(labeledEvent))
			},
		},
		{
			name:      "shard name is not the empty string",
			shardName: testShardName,
			assertions: func(t *testing.T, pred predicate.Predicate, err error) {
				require.NoError(t, err)
				require.NotNil(t, pred)
				require.False(t, pred.Create(unlabeledEvent))
				require.True(t, pred.Create(labeledEvent))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			pred, err := GetShardPredicate(testCase.shardName)
			testCase.assertions(t, pred, err)
		})
	}
}

func TestGetCredentialsRequirement(t *testing.T) {
	testCases := []struct {
		name    string
		labels  labels.Set
		matches bool
	}{
		{
			name: "credential type label set to git",
			labels: labels.Set{
				kargoapi.CredentialTypeLabelKey: credentials.TypeGit.String(),
			},
			matches: true,
		},
		{
			name: "credential type label set to helm and other labels",
			labels: labels.Set{
				kargoapi.CredentialTypeLabelKey: credentials.TypeHelm.String(),
				"other":                         "label",
			},
			matches: true,
		},
		{
			name: "credential type label set to image",
			labels: labels.Set{
				kargoapi.CredentialTypeLabelKey: credentials.TypeImage.String(),
			},
			matches: true,
		},
		{
			name: "credential type label set to unknown type",
			labels: labels.Set{
				kargoapi.CredentialTypeLabelKey: "unknown",
			},
			matches: false,
		},
		{
			name: "with other labels but no credential type label",
			labels: labels.Set{
				"other":   "label",
				"another": "label",
			},
			matches: false,
		},
		{
			name:    "no labels",
			labels:  nil,
			matches: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := GetCredentialsRequirement()
			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, testCase.matches, got.Matches(testCase.labels))
		})
	}
}
