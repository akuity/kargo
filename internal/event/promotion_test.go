package event

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/akuity/kargo/api/v1alpha1"
)

func TestNewPromotionAnnotations(t *testing.T) {
	testCases := map[string]struct {
		actor        string
		promotion    *v1alpha1.Promotion
		freight      *v1alpha1.Freight
		expected     map[string]string
		expectedFunc func(t *testing.T, result map[string]string)
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
				v1alpha1.AnnotationKeyEventProject:             "test-namespace",
				v1alpha1.AnnotationKeyEventPromotionName:       "test-promotion",
				v1alpha1.AnnotationKeyEventFreightName:         "test-freight",
				v1alpha1.AnnotationKeyEventStageName:           "test-stage",
				v1alpha1.AnnotationKeyEventPromotionCreateTime: "2024-10-22T00:00:00Z",
				v1alpha1.AnnotationKeyEventActor:               "promotion-creator",
				v1alpha1.AnnotationKeyEventFreightCreateTime:   "2024-10-22T00:00:00Z",
				v1alpha1.AnnotationKeyEventFreightAlias:        "test-alias",
				v1alpha1.AnnotationKeyEventFreightCommits:      `[{"tag":"test-tag"}]`,
				v1alpha1.AnnotationKeyEventFreightImages:       `[{"tag":"test-tag"}]`,
				v1alpha1.AnnotationKeyEventFreightCharts:       `[{"name":"test-chart"}]`,
				v1alpha1.AnnotationKeyEventApplications: `[{"name":"test-app-1","namespace":"argocd"},` +
					`{"name":"test-app-2","namespace":"test-namespace"}]`,
			},
		},
		"promotion with template variables in argocd app names": {
			actor: "test-user",
			promotion: &v1alpha1.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-promotion",
					Namespace: "kargo-demo",
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
					},
					Annotations: map[string]string{
						v1alpha1.AnnotationKeyCreateActor: "admin",
					},
				},
				Spec: v1alpha1.PromotionSpec{
					Freight: "test-freight",
					Stage:   "test",
					Vars: []v1alpha1.ExpressionVariable{
						{
							Name:  "argocdApp",
							Value: "my-application",
						},
						{
							Name:  "appNamespace",
							Value: "test-namespace",
						},
					},
					Steps: []v1alpha1.PromotionStep{
						{
							Uses: "argocd-update",
							Config: &v1.JSON{Raw: []byte(`{
  "apps": [
    {
      "name": "kargo-demo-${{ ctx.stage }}"
    },
    {
      "name": "${{ vars.argocdApp }}",
      "namespace": "argocd"
    },
    {
      "name": "${{ vars.argocdApp }}-${{ ctx.stage }}",
      "namespace": "${{ vars.appNamespace }}"
    },
    {
      "name": "${{ ctx.promotion }}-${{ ctx.meta.promotion.actor }}",
      "namespace": "${{ ctx.targetFreight.name }}-${{ ctx.targetFreight.origin.name }}"
    }
  ]
}`)},
						},
					},
				},
			},
			freight: &v1alpha1.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-freight",
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
					},
				},
				Origin: v1alpha1.FreightOrigin{
					Name: "test-warehouse",
				},
				Alias: "test-alias",
			},
			expected: map[string]string{
				v1alpha1.AnnotationKeyEventProject:             "kargo-demo",
				v1alpha1.AnnotationKeyEventPromotionName:       "test-promotion",
				v1alpha1.AnnotationKeyEventFreightName:         "test-freight",
				v1alpha1.AnnotationKeyEventFreightAlias:        "test-alias",
				v1alpha1.AnnotationKeyEventStageName:           "test",
				v1alpha1.AnnotationKeyEventPromotionCreateTime: "2024-10-22T00:00:00Z",
				v1alpha1.AnnotationKeyEventActor:               "admin",
				v1alpha1.AnnotationKeyEventFreightCreateTime:   "2024-10-22T00:00:00Z",
				v1alpha1.AnnotationKeyEventApplications: `[{"name":"kargo-demo-test","namespace":"argocd"},` +
					`{"name":"my-application","namespace":"argocd"},` +
					`{"name":"my-application-test","namespace":"test-namespace"},` +
					`{"name":"test-promotion-admin","namespace":"test-freight-test-warehouse"}]`,
			},
		},
		"template evaluation failure - graceful failing": {
			actor: "test-user",
			promotion: &v1alpha1.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-promotion",
					Namespace: "test-namespace",
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
					},
				},
				Spec: v1alpha1.PromotionSpec{
					Freight: "test-freight",
					Stage:   "test-stage",
					Steps: []v1alpha1.PromotionStep{
						{
							Uses: "argocd-update",
							Config: &v1.JSON{Raw: []byte(`{
  "apps": [{"name": "${{ invalid.template.syntax }}"}]
}`)},
						},
					},
				},
			},
			freight: &v1alpha1.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-freight",
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
					},
				},
				Origin: v1alpha1.FreightOrigin{
					Name: "test-warehouse",
				},
			},
			expectedFunc: func(t *testing.T, result map[string]string) {
				require.NotContains(t, result, v1alpha1.AnnotationKeyEventApplications)
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := NewPromotionAnnotations(context.TODO(), tc.actor, tc.promotion, tc.freight)
			if tc.expectedFunc != nil {
				tc.expectedFunc(t, result)
				return
			}
			require.Equal(t, len(tc.expected), len(result), "Number of annotations doesn't match")
			for key, expectedValue := range tc.expected {
				if key == v1alpha1.AnnotationKeyEventApplications {
					expectedAppsJSON := tc.expected[v1alpha1.AnnotationKeyEventApplications]
					actualAppsJSON := result[v1alpha1.AnnotationKeyEventApplications]
					var expectedApps, actualApps []types.NamespacedName
					err := json.Unmarshal([]byte(expectedAppsJSON), &expectedApps)
					require.NoError(t, err, "Failed to unmarshal expected applications")
					err = json.Unmarshal([]byte(actualAppsJSON), &actualApps)
					require.NoError(t, err, "Failed to unmarshal actual applications")
					require.Equal(t, expectedApps, actualApps, "Applications mismatch")
					continue
				}
				actualValue, exists := result[key]
				require.True(t, exists, "Expected annotation %s not found", key)
				require.Equal(t, expectedValue, actualValue, "Annotation %s value mismatch", key)
			}
		})
	}
}
