package event

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/akuity/kargo/api/v1alpha1"
)

func TestNewPromotionEventAnnotations(t *testing.T) {
	testCases := map[string]struct {
		actor     string
		promotion *v1alpha1.Promotion
		freight   *v1alpha1.Freight
		expected  map[string]string
	}{
		"promotion with freight and argocd apps": {
			actor: "test-user",
			promotion: &v1alpha1.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-promotion",
					Namespace: "test-namespace",
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
					},
					Annotations: map[string]string{
						v1alpha1.AnnotationKeyCreateActor: "promotion-creator",
					},
				},
				Spec: v1alpha1.PromotionSpec{
					Freight: "test-freight",
					Stage:   "test-stage",
					Steps: []v1alpha1.PromotionStep{
						{
							Uses: "argocd-update",
							Config: &v1.JSON{Raw: []byte(`{
  "apps": [
    {
      "name": "test-app-1"
    },
    {
      "name": "test-app-2",
      "namespace": "test-namespace"
    }
  ]
}`)},
						},
					},
				},
			},
			freight: &v1alpha1.Freight{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
					},
				},
				Alias:   "test-alias",
				Commits: []v1alpha1.GitCommit{{Tag: "test-tag"}},
				Images:  []v1alpha1.Image{{Tag: "test-tag"}},
				Charts:  []v1alpha1.Chart{{Name: "test-chart"}},
			},
			expected: map[string]string{
				AnnotationKeyEventProject:             "test-namespace",
				AnnotationKeyEventPromotionName:       "test-promotion",
				AnnotationKeyEventFreightName:         "test-freight",
				AnnotationKeyEventStageName:           "test-stage",
				AnnotationKeyEventPromotionCreateTime: "2024-10-22T00:00:00Z",
				AnnotationKeyEventActor:               "promotion-creator",
				AnnotationKeyEventFreightCreateTime:   "2024-10-22T00:00:00Z",
				AnnotationKeyEventFreightAlias:        "test-alias",
				AnnotationKeyEventFreightCommits:      `[{"tag":"test-tag"}]`,
				AnnotationKeyEventFreightImages:       `[{"tag":"test-tag"}]`,
				AnnotationKeyEventFreightCharts:       `[{"name":"test-chart"}]`,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := NewPromotionEventAnnotations(context.TODO(), tc.actor, tc.promotion, tc.freight)

			require.Equal(t, len(tc.expected), len(result), "Number of annotations doesn't match")
			for key, expectedValue := range tc.expected {
				actualValue, exists := result[key]
				require.True(t, exists, "Expected annotation %s not found", key)
				require.Equal(t, expectedValue, actualValue, "Annotation %s value mismatch", key)
			}
		})
	}
}
