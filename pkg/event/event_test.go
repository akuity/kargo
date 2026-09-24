package event

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewCommonFromPromotion(t *testing.T) {
	testCases := map[string]struct {
		message   string
		actor     string
		promotion *kargoapi.Promotion
		expected  Common
	}{
		"promotion with actor annotation": {
			message: "test message",
			actor:   "external-actor",
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-project",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyCreateActor: "promotion-actor",
					},
				},
			},
			expected: Common{
				Project: "test-project",
				Message: "test message",
				Actor:   ptr.To("promotion-actor"), // annotation takes precedence
			},
		},
		"promotion without actor annotation": {
			message: "test message",
			actor:   "external-actor",
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-project",
				},
			},
			expected: Common{
				Project: "test-project",
				Message: "test message",
				Actor:   ptr.To("external-actor"),
			},
		},
		"nil promotion": {
			message:   "test message",
			actor:     "external-actor",
			promotion: nil,
			expected:  Common{},
		},
		"empty actor": {
			message: "test message",
			actor:   "",
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-project",
				},
			},
			expected: Common{
				Project: "test-project",
				Message: "test message",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := newCommonFromPromotion(tc.message, tc.actor, tc.promotion)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestNewCommonFromFreight(t *testing.T) {
	testCases := map[string]struct {
		message  string
		actor    string
		freight  *kargoapi.Freight
		expected Common
	}{
		"complete freight": {
			message: "test message",
			actor:   "test-actor",
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-project",
				},
			},
			expected: Common{
				Project: "test-project",
				Message: "test message",
				Actor:   ptr.To("test-actor"),
			},
		},
		"empty actor": {
			message: "test message",
			actor:   "",
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-project",
				},
			},
			expected: Common{
				Project: "test-project",
				Message: "test message",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := newCommonFromFreight(tc.message, tc.actor, tc.freight)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestNewFreight(t *testing.T) {
	testCases := map[string]struct {
		freight   *kargoapi.Freight
		stageName string
		expected  Freight
	}{
		"complete freight": {
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-freight",
					CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
				Alias:   "v1.0.0",
				Commits: []kargoapi.GitCommit{{ID: "abc123", Tag: "v1.0.0"}},
				Images:  []kargoapi.Image{{RepoURL: "example.com/app", Tag: "v1.0.0"}},
				Charts:  []kargoapi.Chart{{Name: "my-chart", Version: "1.0.0"}},
				Artifacts: []kargoapi.ArtifactReference{
					{ArtifactType: "my-type", SubscriptionName: "my-sub", Version: "v1.0.0"},
				},
			},
			stageName: "test-stage",
			expected: Freight{
				CreateTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Name:       "test-freight",
				StageName:  "test-stage",
				Alias:      ptr.To("v1.0.0"),
				Commits:    []kargoapi.GitCommit{{ID: "abc123", Tag: "v1.0.0"}},
				Images:     []kargoapi.Image{{RepoURL: "example.com/app", Tag: "v1.0.0"}},
				Charts:     []kargoapi.Chart{{Name: "my-chart", Version: "1.0.0"}},
				Artifacts: []kargoapi.ArtifactReference{
					{ArtifactType: "my-type", SubscriptionName: "my-sub", Version: "v1.0.0"},
				},
			},
		},
		"minimal freight": {
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-freight",
					CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			},
			stageName: "test-stage",
			expected: Freight{
				CreateTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Name:       "test-freight",
				StageName:  "test-stage",
			},
		},
		"nil freight": {
			freight:   nil,
			stageName: "test-stage",
			expected:  Freight{},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := newFreight(tc.freight, tc.stageName)
			require.Equal(t, tc.expected, result)
		})
	}
}
